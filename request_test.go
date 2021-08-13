package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
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

	c.Request().Header.SetCookie("session_id", "abcdef123")
	c.Request().Header.SetCookie("ui_id", "123")

	cookieJar, err := cookiejar.New(nil)
	require.NoError(t, err)
	c.Locals("cookieJar", cookieJar)
	reqURL, err := url.Parse("https://www.google.com")
	require.NoError(t, err)
	cookieJar.SetCookies(reqURL, []*http.Cookie{{Name: "google_sid", Value: "123"}})

	p := NewRequestProcessor(NewDomainConverter("example.com"))
	result := p.Process(c)
	require.Equal(t, "www.google.com", result.URL.Host)
	require.Equal(t, "/abc", result.URL.Path)
	require.Equal(t, "q=1", result.URL.RawQuery)
	require.Equal(t, []string{"https://www.google.com/def"}, result.Header["Referer"])
	require.Equal(t, []string{"https://www.google.com"}, result.Header["Origin"])

	cookies := result.Cookies()
	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Name < cookies[j].Name
	})
	require.Equal(t, []*http.Cookie{
		{Name: "google_sid", Value: "123"},
		{Name: "ui_id", Value: "123"},
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

	p := NewRequestProcessor(NewDomainConverter("example.com"))
	result := p.Process(c)
	require.Equal(t, "www.google.com", result.URL.Host)
	require.Equal(t, "/abc", result.URL.Path)
	require.Equal(t, "q=1", result.URL.RawQuery)
	require.Equal(t, []string{"https://www.google.com/def"}, result.Header["Referer"])
	require.Equal(t, []string{"https://www.google.com"}, result.Header["Origin"])

	require.Empty(t, result.Cookies())
	require.Nil(t, result.Body)
}
