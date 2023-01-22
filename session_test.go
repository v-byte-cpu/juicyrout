package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestSessionManagerNewSession(t *testing.T) {
	store := session.New()
	sm := NewSessionManager(store, sessionCookieName)

	app := fiber.New()
	fiberCtx0 := app.AcquireCtx(&fasthttp.RequestCtx{})
	fiberCtx0.Request().Header.SetCookie("session_id", "session1")
	sess, err := sm.GetSession(fiberCtx0)
	require.NoError(t, err)
	require.Nil(t, sess)

	lureURL := "/abc/def"
	fiberCtx1 := app.AcquireCtx(&fasthttp.RequestCtx{})
	sess1, err := sm.NewSession(fiberCtx1, lureURL)
	require.NoError(t, err)
	require.NotNil(t, sess1)

	u, err := url.Parse("https://google.com")
	require.NoError(t, err)
	expectedCookies := []*http.Cookie{{Name: "abc", Value: "def"}}
	sess1.CookieJar().SetCookies(u, expectedCookies)
	session1ID := sess1.ID()
	cookieJar1 := sess1.CookieJar()
	require.NoError(t, sess1.Save())

	fiberCtx := app.AcquireCtx(&fasthttp.RequestCtx{})
	fiberCtx.Request().Header.SetCookie("session_id", session1ID)
	sess1new, err := sm.GetSession(fiberCtx)
	require.NoError(t, err)
	require.NotNil(t, sess1new)
	require.True(t, cookieJar1 == sess1new.CookieJar(), "Cookie Jars are not the same")
	cookies := sess1new.CookieJar().Cookies(u)
	require.Equal(t, expectedCookies, cookies)

	fiberCtx2 := app.AcquireCtx(&fasthttp.RequestCtx{})
	sess2, err := sm.NewSession(fiberCtx2, lureURL)
	require.NoError(t, err)
	require.NotNil(t, sess2)
	require.True(t, sess2 != sess1, "sessions are the same")
}

func TestSessionManagerDelete(t *testing.T) {
	store := session.New()
	sm := NewSessionManager(store, sessionCookieName)
	app := fiber.New()
	fiberCtx1 := app.AcquireCtx(&fasthttp.RequestCtx{})
	lureURL := "/abc/def"
	sess1, err := sm.NewSession(fiberCtx1, lureURL)
	require.NoError(t, err)

	u, err := url.Parse("https://google.com")
	require.NoError(t, err)
	cookies := []*http.Cookie{{Name: "abc", Value: "def"}}
	sess1.CookieJar().SetCookies(u, cookies)

	sm.DeleteSession(sess1.ID())
	fiberCtx := app.AcquireCtx(&fasthttp.RequestCtx{})
	fiberCtx.Request().Header.SetCookie("session_id", sess1.ID())
	sess, err := sm.GetSession(fiberCtx)
	require.NoError(t, err)
	require.Nil(t, sess)

	fiberCtx1new := app.AcquireCtx(&fasthttp.RequestCtx{})
	sess1new, err := sm.NewSession(fiberCtx1new, lureURL)
	require.NoError(t, err)
	require.NotNil(t, sess1new)
	require.NotEqual(t, sess1new, sess1, "session objects are the same")
}

func TestNewSessionStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	sm := NewMockSessionManager(ctrl)
	s := NewSessionStorage(memory.New(), sm)

	data := []byte("def")
	err := s.Set("abc", data, 0)
	require.NoError(t, err)

	v, err := s.Get("abc")
	require.NoError(t, err)
	require.Equal(t, data, v)

	sm.EXPECT().DeleteSession("abc")

	err = s.Delete("abc")
	require.NoError(t, err)
}
