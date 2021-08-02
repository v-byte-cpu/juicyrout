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
	c.Request().Header.Add("Referer", "https://abc.com")
	c.Request().Header.Add("Origin", "https://www-google-com.example.com")

	p := NewRequestProcessor(NewDomainConverter("example.com"))
	result := p.Process(c)
	require.Equal(t, "www.google.com", result.URL.Host)
	require.Equal(t, "/abc", result.URL.Path)
	require.Equal(t, "q=1", result.URL.RawQuery)
	require.Zero(t, len(result.Header["Referer"]))
	require.Equal(t, []string{"https://www.google.com"}, result.Header["Origin"])
	require.Nil(t, result.Body)
}
