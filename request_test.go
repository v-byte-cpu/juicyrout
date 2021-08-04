package main

import (
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

	cookie := &fasthttp.Cookie{}
	cookie.SetKey("session_id")
	cookie.SetValue("abcdef123")
	cookie.SetDomain("example.com")
	cookie.SetSecure(true)
	c.Request().Header.SetCookieBytesKV(cookie.Key(), cookie.AppendBytes(nil))

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
