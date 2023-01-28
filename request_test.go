package main

import (
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestRequestProcessor(t *testing.T) {
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	c.Request().Header.SetMethod(fasthttp.MethodGet)
	c.Request().SetRequestURI("https://www-google-com.example.com/abc?q=1")
	c.Request().Header.Add("Referer", "https://www-google-com.example.com/def")
	c.Request().Header.Add("Origin", "https://www-google-com.example.com")
	c.Request().Header.Add("Host", "www-google-com.example.com")

	c.Request().Header.SetCookie("ui_id", "123")
	logger := zerolog.New(io.Discard)
	sm := NewSessionManager(&logger, session.New(), sessionCookieName)
	sess, err := sm.NewSession(c, "/abc/def")
	require.NoError(t, err)

	c.Request().Header.SetCookie("session_id", sess.ID())
	setProxySession(c, sess)

	reqURL, err := url.Parse("https://www.google.com")
	require.NoError(t, err)
	sess.CookieJar().SetCookies(reqURL, []*http.Cookie{{Name: "google_sid", Value: "123"}})

	p := newRequestProcessor("example.com")
	result := p.Process(c)
	require.Equal(t, "www.google.com", result.URL.Host)
	require.Equal(t, "/abc", result.URL.Path)
	require.Equal(t, "q=1", result.URL.RawQuery)
	require.Equal(t, []string{"https://www.google.com/def"}, result.Header["Referer"])
	require.Equal(t, []string{"https://www.google.com"}, result.Header["Origin"])
	require.Equal(t, []string{"www.google.com"}, result.Header["Host"])

	cookies := result.Cookies()
	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Name < cookies[j].Name
	})
	require.Equal(t, []*http.Cookie{
		{Name: "google_sid", Value: "123"},
	}, cookies)

	require.Nil(t, result.Body)
}

func TestRequestProcessorOptionsMethod(t *testing.T) {
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	c.Request().Header.SetMethod(fasthttp.MethodOptions)
	c.Request().SetRequestURI("https://www-google-com.example.com/abc?q=1")
	c.Request().Header.Add("Referer", "https://www-google-com.example.com/def")
	c.Request().Header.Add("Origin", "https://www-google-com.example.com")

	p := newRequestProcessor("example.com")
	result := p.Process(c)
	require.Equal(t, "www.google.com", result.URL.Host)
	require.Equal(t, "/abc", result.URL.Path)
	require.Equal(t, "q=1", result.URL.RawQuery)
	require.Equal(t, []string{"https://www.google.com/def"}, result.Header["Referer"])
	require.Equal(t, []string{"https://www.google.com"}, result.Header["Origin"])

	require.Empty(t, result.Cookies())
	require.Nil(t, result.Body)
}

func TestRequestProcessorModifyQuery(t *testing.T) {
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	logger := zerolog.New(io.Discard)
	sm := NewSessionManager(&logger, session.New(), sessionCookieName)
	sess, err := sm.NewSession(c, "/abc/def")
	require.NoError(t, err)
	setProxySession(c, sess)

	c.Request().Header.SetMethod(fasthttp.MethodGet)
	c.Request().SetRequestURI("https://www-google-com.example.com/abc?q=https%3A%2f%2Fgoogle-com.example.com&hash=ABCdef")
	p := newRequestProcessor("example.com")
	result := p.Process(c)

	require.Equal(t, "q=https%3A%2f%2fgoogle.com&hash=ABCdef", result.URL.RawQuery)
}

func TestRequestProcessorModifyBody(t *testing.T) {
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	logger := zerolog.New(io.Discard)
	sm := NewSessionManager(&logger, session.New(), sessionCookieName)
	sess, err := sm.NewSession(c, "/abc/def")
	require.NoError(t, err)
	setProxySession(c, sess)

	c.Request().Header.SetMethod(fasthttp.MethodGet)
	c.Request().SetRequestURI("https://www-google-com.example.com")
	c.Request().SetBodyStream(strings.NewReader(`{"url":"https://www-google-com.example.com"}`), -1)

	p := newRequestProcessor("example.com")
	result := p.Process(c)

	defer result.Body.Close()
	data, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.Equal(t, `{"url":"https://www.google.com"}`, string(data))
}

func TestRequestProcessorSaveUserAgent(t *testing.T) {
	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)

	logger := zerolog.New(io.Discard)
	sm := NewSessionManager(&logger, session.New(), sessionCookieName)
	sess, err := sm.NewSession(c, "/abc/def")
	require.NoError(t, err)
	setProxySession(c, sess)
	const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36"

	c.Request().Header.SetMethod(fasthttp.MethodGet)
	c.Request().SetRequestURI("https://www-google-com.example.com/abc?q=1")
	c.Request().Header.Add("User-Agent", userAgent)

	p := newRequestProcessor("example.com")
	cnt := 0
	p.userAgentSaver = mockUserAgentSaverFunc(func(c *fiber.Ctx) {
		cnt++
		require.Equal(t, userAgent, c.Get("User-Agent"))
	})
	p.Process(c)

	require.Equal(t, 1, cnt, "userAgentSaver called invalid number of times")

}

//nolint:unparam
func newRequestProcessor(domain string) *requestProcessor {
	logger := zerolog.New(io.Discard)
	conv := NewDomainConverter(domain)
	urlProc := newURLRegexProcessor(&logger, func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	userAgentSaver := mockUserAgentSaverFunc(func(c *fiber.Ctx) {})
	return NewRequestProcessor(conv, urlProc, userAgentSaver).(*requestProcessor)
}

type mockUserAgentSaverFunc func(c *fiber.Ctx)

func (f mockUserAgentSaverFunc) SaveUserAgent(c *fiber.Ctx) {
	f(c)
}
