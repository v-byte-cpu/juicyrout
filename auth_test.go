package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	invalidAuthURL = "https://google.com/hackz"
	loginURL       = "https://google-com.example.com/login"
)

type mockLureService struct {
	lures map[string]struct{}
}

func (s *mockLureService) ExistsByURL(url string) (bool, error) {
	_, ok := s.lures[url]
	return ok, nil
}

func (*mockLureService) Add(_ *APILure) error {
	return nil
}

func (*mockLureService) DeleteByURL(_ string) error {
	return nil
}

func (*mockLureService) GetAll() ([]*APILure, error) {
	return nil, nil
}

func TestAuthMiddleware(t *testing.T) {

	notCalled := func(c *fiber.Ctx) error {
		require.Fail(t, "should not be called")
		return nil
	}

	tests := []struct {
		name               string
		handler            fiber.Handler
		inputURL           string
		redirectURL        string
		sessionCookie      bool
		inputValidCookie   bool
		inputInvalidCookie bool
		data               bool
	}{
		{
			name:          "NoCookiesInvalidURL",
			handler:       notCalled,
			inputURL:      "/invalid_lure_url",
			redirectURL:   invalidAuthURL,
			sessionCookie: false,
		},
		{
			name:          "NoCookiesValidURL",
			handler:       notCalled,
			inputURL:      "/abc/def",
			redirectURL:   loginURL,
			sessionCookie: true,
		},
		{
			name:               "InvalidCookiesInvalidURL",
			handler:            notCalled,
			inputURL:           "/invalid_lure_url",
			redirectURL:        invalidAuthURL,
			sessionCookie:      false,
			inputInvalidCookie: true,
		},
		{
			name:               "InvalidCookiesValidURL",
			handler:            notCalled,
			inputURL:           "/abc/def",
			redirectURL:        loginURL,
			sessionCookie:      true,
			inputInvalidCookie: true,
		},
		{
			name: "ValidCookiesLoginURL",
			handler: func(c *fiber.Ctx) error {
				sess := getSession(c)
				require.Equal(t, "/abc/def", sess.Get("lureURL").(string))

				cookieJar := getCookieJar(c)
				require.NotNil(t, cookieJar)
				return c.SendString("Hello")
			},
			inputURL:         "/login",
			inputValidCookie: true,
			data:             true,
		},
		{
			name:             "ValidCookiesLureURL",
			handler:          notCalled,
			inputURL:         "/abc/def",
			redirectURL:      loginURL,
			inputValidCookie: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			app.Use(NewAuthMiddleware(AuthConfig{
				CookieName:    "session_id",
				CookieManager: NewCookieManager(),
				Store: session.New(session.Config{
					KeyLookup:    "cookie:session_id",
					CookieDomain: "example.com",
				}),
				InvalidAuthURL: invalidAuthURL,
				LoginURL:       loginURL,
				LureService: &mockLureService{lures: map[string]struct{}{
					"/abc/def": {},
				}},
			}))
			app.All("/*", tt.handler)

			req := httptest.NewRequest("GET", tt.inputURL, nil)
			if tt.inputInvalidCookie {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
			} else if tt.inputValidCookie {
				req.AddCookie(getValidCookie(t, app))
			}

			resp, err := app.Test(req)
			require.NoError(t, err)

			if len(tt.redirectURL) > 0 {
				assert.Equal(t, http.StatusFound, resp.StatusCode)
				require.NotZero(t, len(resp.Header["Location"]))
				require.NotZero(t, len(resp.Header["Referrer-Policy"]))
				require.Equal(t, tt.redirectURL, resp.Header["Location"][0])
				require.Equal(t, "no-referrer", resp.Header["Referrer-Policy"][0])
			}

			if tt.data {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				require.Zero(t, len(resp.Header["Location"]))
				require.Zero(t, len(resp.Header["Referrer-Policy"]))

				data, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Equal(t, "Hello", string(data))
			}

			if tt.sessionCookie {
				require.NotEmpty(t, resp.Cookies())
				require.Equal(t, "session_id", resp.Cookies()[0].Name)
				require.Equal(t, "example.com", resp.Cookies()[0].Domain)
				require.NotEmpty(t, resp.Cookies()[0].Value)
			} else {
				require.Empty(t, resp.Cookies())
			}
		})
	}
}

func TestAuthMiddlewareNilCookieJar(t *testing.T) {
	cm := NewCookieManager()
	app := fiber.New()
	app.Use(NewAuthMiddleware(AuthConfig{
		CookieName:    "session_id",
		CookieManager: cm,
		Store: session.New(session.Config{
			KeyLookup:    "cookie:session_id",
			CookieDomain: "example.com",
		}),
		InvalidAuthURL: invalidAuthURL,
		LoginURL:       "https://google-com.example.com/login",
		LureService: &mockLureService{lures: map[string]struct{}{
			"/abc/def": {},
		}},
	}))
	app.All("/*", func(c *fiber.Ctx) error {
		require.Fail(t, "should not be called")
		return nil
	})

	req := httptest.NewRequest("GET", "/some/valid/url", nil)
	cookie := getValidCookie(t, app)
	req.AddCookie(cookie)

	cm.Delete(cookie.Value)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	require.NotZero(t, len(resp.Header["Location"]))
	require.NotZero(t, len(resp.Header["Referrer-Policy"]))
	require.Equal(t, invalidAuthURL, resp.Header["Location"][0])
	require.Equal(t, "no-referrer", resp.Header["Referrer-Policy"][0])
}

func getValidCookie(t *testing.T, app *fiber.App) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest("GET", "/abc/def", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.NotZero(t, len(resp.Cookies()))
	cookie := resp.Cookies()[0]
	require.Equal(t, "session_id", cookie.Name)
	return cookie
}
