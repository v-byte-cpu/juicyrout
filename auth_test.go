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
	invalidAuthURL    = "https://google.com/hackz"
	loginURL          = "https://google-com.example.com/login"
	sessionCookieName = "session_id"
)

type mockLureService struct {
	lures map[string]*APILure
}

func (s *mockLureService) ExistsByURL(url string) (bool, error) {
	_, ok := s.lures[url]
	return ok, nil
}

func (s *mockLureService) GetByURL(url string) (*APILure, error) {
	return s.lures[url], nil
}

func (*mockLureService) Add(*APILure) error {
	return nil
}

func (*mockLureService) DeleteByURL(string) error {
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
				sess := getProxySession(c)
				require.Equal(t, "/abc/def", sess.Get("lureURL").(string))

				require.NotNil(t, sess.CookieJar())
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
			store := session.New(session.Config{
				KeyLookup:    "cookie:session_id",
				CookieDomain: "example.com",
			})
			app.Use(NewAuthMiddleware(AuthConfig{
				SessionManager: NewSessionManager(store, sessionCookieName),
				InvalidAuthURL: invalidAuthURL,
				LoginURL:       loginURL,
				LureService: &mockLureService{lures: map[string]*APILure{
					"/abc/def": {LureURL: "/abc/def"},
				}},
				AuthService: authServiceFunc(func(*fiber.Ctx) bool {
					return false
				}),
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
				require.Equal(t, []string{tt.redirectURL}, resp.Header["Location"])
				require.Equal(t, []string{"no-referrer"}, resp.Header["Referrer-Policy"])
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

func TestAuthMiddlewareAuthenticatedLoginURL(t *testing.T) {
	app := fiber.New()
	store := session.New(session.Config{
		KeyLookup:    "cookie:session_id",
		CookieDomain: "example.com",
	})
	app.Use(NewAuthMiddleware(AuthConfig{
		SessionManager: NewSessionManager(store, sessionCookieName),
		InvalidAuthURL: invalidAuthURL,
		LoginURL:       loginURL,
		LureService: &mockLureService{lures: map[string]*APILure{
			"/abc/def": {LureURL: "/abc/def", TargetURL: "/target/url"},
		}},
		AuthService: authServiceFunc(func(*fiber.Ctx) bool {
			return true
		}),
	}))
	app.All("/*", func(c *fiber.Ctx) error {
		require.Fail(t, "should not be called")
		return nil
	})

	req := httptest.NewRequest("GET", "/abc/def", nil)
	cookie := getValidCookie(t, app)
	req.AddCookie(cookie)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	require.NotZero(t, len(resp.Header["Referrer-Policy"]))
	require.Equal(t, []string{"/target/url"}, resp.Header["Location"])
	require.Equal(t, []string{"no-referrer"}, resp.Header["Referrer-Policy"])
}

func TestAuthMiddlewareNilCookieJar(t *testing.T) {
	store := session.New(session.Config{
		KeyLookup:    "cookie:session_id",
		CookieDomain: "example.com",
	})
	sm := NewSessionManager(store, sessionCookieName)
	app := fiber.New()
	app.Use(NewAuthMiddleware(AuthConfig{
		SessionManager: sm,
		InvalidAuthURL: invalidAuthURL,
		LoginURL:       loginURL,
		LureService: &mockLureService{lures: map[string]*APILure{
			"/abc/def": {LureURL: "/abc/def"},
		}},
	}))
	app.All("/*", func(c *fiber.Ctx) error {
		require.Fail(t, "should not be called")
		return nil
	})

	req := httptest.NewRequest("GET", "/some/valid/url", nil)
	cookie := getValidCookie(t, app)
	req.AddCookie(cookie)

	sm.DeleteSession(cookie.Value)

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, []string{invalidAuthURL}, resp.Header["Location"])
	require.Equal(t, []string{"no-referrer"}, resp.Header["Referrer-Policy"])
}

func getValidCookie(t *testing.T, app *fiber.App) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest("GET", "/abc/def", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.NotZero(t, len(resp.Cookies()))
	cookie := resp.Cookies()[0]
	require.Equal(t, "session_id", cookie.Name)
	return cookie
}
