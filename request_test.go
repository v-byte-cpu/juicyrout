package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
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

	c.Request().Header.SetCookie("session_id", "abcdef123")
	c.Request().Header.SetCookie("ui_id", "123")

	cookieJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	setCookieJar(c, cookieJar)
	reqURL, err := url.Parse("https://www.google.com")
	require.NoError(t, err)
	cookieJar.SetCookies(reqURL, []*http.Cookie{{Name: "google_sid", Value: "123"}})

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

	cookieJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	setCookieJar(c, cookieJar)

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

	cookieJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	setCookieJar(c, cookieJar)

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

//nolint:unparam
func newRequestProcessor(domain string) *requestProcessor {
	conv := NewDomainConverter(domain)
	urlProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	return NewRequestProcessor(conv, urlProc).(*requestProcessor)
}
