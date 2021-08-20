package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/valyala/fasthttp"
)

type APIConfig struct {
	// Hostname for API endpoints
	APIHostname string
	// DomainConverter used to convert proxy domain to target original domain
	DomainConverter DomainConverter
}

func NewAPIMiddleware(conf APIConfig) fiber.Handler {
	m := &apiMiddleware{&conf}
	api := fiber.New()
	api.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))
	api.Get("/cookies", func(c *fiber.Ctx) error {
		return m.GetCookies(c)
	})
	api.Post("/cookies", func(c *fiber.Ctx) error {
		return m.CreateCookie(c)
	})
	apiHandler := api.Handler()
	return func(c *fiber.Ctx) error {
		if c.Hostname() != conf.APIHostname {
			return c.Next()
		}
		apiHandler(c.Context())
		return nil
	}
}

type apiMiddleware struct {
	*APIConfig
}

func (m *apiMiddleware) GetCookies(c *fiber.Ctx) error {
	destURL, err := m.getOrigin(c)
	if err != nil {
		return err
	}
	cookieJar := c.Locals("cookieJar").(http.CookieJar)
	cookies := cookieJar.Cookies(destURL)
	cookieStr := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		cookieStr = append(cookieStr, cookie.String())
	}
	return c.SendString(strings.Join(cookieStr, "; "))
}

func (m *apiMiddleware) CreateCookie(c *fiber.Ctx) error {
	destURL, err := m.getOrigin(c)
	if err != nil {
		return err
	}
	cookie := &fasthttp.Cookie{}
	if err := cookie.ParseBytes(c.Body()); err != nil {
		return err
	}
	newCookie := &http.Cookie{
		Name:     utils.UnsafeString(cookie.Key()),
		Value:    utils.UnsafeString(cookie.Value()),
		Path:     utils.UnsafeString(cookie.Path()),
		Domain:   utils.UnsafeString(cookie.Domain()),
		Expires:  cookie.Expire(),
		MaxAge:   cookie.MaxAge(),
		Secure:   cookie.Secure(),
		HttpOnly: cookie.HTTPOnly(),
	}
	cookieJar := c.Locals("cookieJar").(http.CookieJar)
	cookieJar.SetCookies(destURL, []*http.Cookie{newCookie})
	return nil
}

func (m *apiMiddleware) getOrigin(c *fiber.Ctx) (destURL *url.URL, err error) {
	origin := m.DomainConverter.ToTargetURL(c.Get("Origin"))
	destURL, err = url.Parse(origin)
	return
}