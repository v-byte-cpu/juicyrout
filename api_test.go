package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestAPIMiddlewareNextForwarding(t *testing.T) {
	app := fiber.New()
	app.Use(NewAPIMiddleware(APIConfig{APIHostname: "api.example.com"}))
	app.All("/*", func(c *fiber.Ctx) error {
		return c.SendString("Hello")
	})

	req := httptest.NewRequest("GET", "https://google-com.example.com/abc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello", string(data))
}

func TestAPIMiddlewareGetCookies(t *testing.T) {

	tests := []struct {
		name     string
		cookies  []*http.Cookie
		expected []*http.Cookie
	}{
		{
			name: "OneCookie",
			cookies: []*http.Cookie{
				{Name: "abc", Value: "def", Domain: "google.com"},
			},
			expected: []*http.Cookie{
				{Name: "abc", Value: "def"},
			},
		},
		{
			name: "TwoCookies",
			cookies: []*http.Cookie{
				{Name: "abc", Value: "def", Domain: "google.com"},
				{Name: "abc2", Value: "def2", Domain: "www.google.com"},
			},
			expected: []*http.Cookie{
				{Name: "abc", Value: "def"},
				{Name: "abc2", Value: "def2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cm := NewCookieManager()
			app := createAPIApp(t, cm)

			req := httptest.NewRequest("GET", "https://api.example.com/cookies", nil)
			cookie := getValidCookie(t, app)
			req.AddCookie(cookie)
			req.Header["Origin"] = []string{"https://www-google-com.example.com"}

			cookieJar := cm.Get(cookie.Value)
			cookieJar.SetCookies(&url.URL{Scheme: "https", Host: "www.google.com"}, tt.cookies)

			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			cookies := parseCookies(string(data))
			require.Equal(t, tt.expected, cookies)
		})
	}
}

func TestAPIMiddlewareCreateCookies(t *testing.T) {
	tests := []struct {
		name      string
		cookies   []*http.Cookie
		newCookie *http.Cookie
		expected  []*http.Cookie
	}{
		{
			name:    "OneCookie",
			cookies: []*http.Cookie{},
			newCookie: &http.Cookie{
				Name: "abc", Value: "def", Domain: "google.com",
			},
			expected: []*http.Cookie{
				{Name: "abc", Value: "def"},
			},
		},
		{
			name: "TwoCookies",
			cookies: []*http.Cookie{
				{Name: "abc", Value: "def", Domain: "google.com"},
			},
			newCookie: &http.Cookie{
				Name: "abc2", Value: "def2", Domain: "www.google.com",
			},
			expected: []*http.Cookie{
				{Name: "abc", Value: "def"},
				{Name: "abc2", Value: "def2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cm := NewCookieManager()
			app := createAPIApp(t, cm)

			body := strings.NewReader(tt.newCookie.String())
			req := httptest.NewRequest("POST", "https://api.example.com/cookies", body)
			cookie := getValidCookie(t, app)
			req.AddCookie(cookie)
			req.Header["Origin"] = []string{"https://www-google-com.example.com"}

			cookieJar := cm.Get(cookie.Value)
			cookieJar.SetCookies(&url.URL{Scheme: "https", Host: "www.google.com"}, tt.cookies)

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// check that cookies have been created
			req = httptest.NewRequest("GET", "https://api.example.com/cookies", nil)
			req.AddCookie(cookie)
			req.Header["Origin"] = []string{"https://www-google-com.example.com"}

			resp, err = app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			cookies := parseCookies(string(data))
			require.Equal(t, tt.expected, cookies)
		})
	}
}

func TestAPIMiddlewareGetOrigin(t *testing.T) {
	api := &apiMiddleware{&APIConfig{
		APIHostname:     "api.example.com",
		DomainConverter: NewDomainConverter("host.juicyrout:8091"),
	}}
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	c.Request().Header.Add("Origin", "https://www-instagram-com.host.juicyrout:8091")

	destURL, err := api.getOrigin(c)
	require.NoError(t, err)
	require.Equal(t, "www.instagram.com", destURL.Host)
	require.Equal(t, "https://www.instagram.com", destURL.String())
}

func createAPIApp(t *testing.T, cm CookieManager) *fiber.App {
	t.Helper()
	app := fiber.New()
	auth := NewAuthMiddleware(AuthConfig{
		CookieName:    "session_id",
		CookieManager: cm,
		Store: session.New(session.Config{
			KeyLookup:    "cookie:session_id",
			CookieDomain: "example.com",
		}),
		InvalidAuthURL: invalidAuthURL,
		LoginURL:       loginURL,
		LureService: &mockLureService{lures: map[string]struct{}{
			"/abc/def": {},
		}},
	})
	api := NewAPIMiddleware(APIConfig{
		APIHostname:     "api.example.com",
		DomainConverter: NewDomainConverter("example.com"),
	})
	app.Use(auth, api)
	app.All("/*", func(c *fiber.Ctx) error {
		require.Fail(t, "should not be called")
		return nil
	})
	return app
}

func parseCookies(data string) []*http.Cookie {
	if len(data) == 0 {
		return nil
	}
	cookieStr := strings.Split(data, "; ")
	cookies := make([]*http.Cookie, 0, len(cookieStr))
	for _, cookie := range cookieStr {
		parts := strings.Split(cookie, "=")
		cookies = append(cookies, &http.Cookie{Name: parts[0], Value: parts[1]})
	}
	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Name < cookies[j].Name
	})
	return cookies
}
