package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
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
			origin: "www-google-com.example.com",
		},
		{
			name:   "WithTargetOrigin",
			origin: "www.google.com",
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
			resp.Header["Access-Control-Allow-Origin"] = []string{"www.google.com"}
			proc.convertCORS(resp)
			origins := resp.Header["Access-Control-Allow-Origin"]
			require.Equal(t, "www-google-com.example.com", origins[0])
			require.Equal(t, "true", resp.Header["Access-Control-Allow-Credentials"][0])
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
			cookieJar, err := cookiejar.New(nil)
			require.NoError(t, err)
			setCookieJar(c, cookieJar)

			proc.writeCookies(c, resp)

			require.Zero(t, len(resp.Header["Set-Cookie"]))

			jarCookies := cookieJar.Cookies(reqURL)
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
		Body:       ioutil.NopCloser(strings.NewReader("")),
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
				Body:   ioutil.NopCloser(strings.NewReader(tt.input)),
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
	return NewResponseProcessor(conv, urlProc, htmlProc).(*responseProcessor)
}
