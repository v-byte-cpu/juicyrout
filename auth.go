package main

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"go.uber.org/multierr"
)

type AuthConfig struct {
	// Name of authentication cookie
	CookieName string
	// Store interface to store the session data
	Store *session.Store
	// URL to redirect browser in case of invalid lure URL or cookies
	InvalidAuthURL string
	// URL to redirect browser in case of valid lure URL and empty cookies
	LoginURL string
	// Service to check URLs for authentication and redirect configs
	LureService LureService
	// Manager to retrieve cookie jar for session
	CookieManager CookieManager
}

func NewAuthMiddleware(conf AuthConfig) fiber.Handler {
	m := &authMiddleware{AuthConfig: &conf}
	return func(c *fiber.Ctx) error {
		return m.Handle(c)
	}
}

type authMiddleware struct {
	*AuthConfig
}

func (m *authMiddleware) Handle(c *fiber.Ctx) error {
	if m.validateCookies(c) {
		exists, err := m.LureService.ExistsByURL(c.OriginalURL())
		if err != nil {
			return err
		}
		if exists {
			// TODO check real auth and forward to lure target URL
			return m.redirect(c, m.LoginURL)
		}

		// retrieve existing session
		sess, err := m.Store.Get(c)
		if err != nil {
			return err
		}
		cookieJar := m.CookieManager.GetSession(sess.ID())
		if cookieJar == nil {
			log.Println("cookieJar is nil!")
			return m.redirect(c, m.InvalidAuthURL)
		}

		setCookieJar(c, cookieJar)
		setSession(c, sess)
		err = c.Next()
		// refresh session
		return multierr.Append(err, sess.Save())
	}
	exists, err := m.LureService.ExistsByURL(c.OriginalURL())
	if err != nil {
		return err
	}
	if !exists {
		log.Println("invalid auth url, request hostname:", c.Hostname())
		return m.redirect(c, m.InvalidAuthURL)
	}
	if err := m.createNewSession(c); err != nil {
		return err
	}
	return m.redirect(c, m.LoginURL)
}

func (*authMiddleware) redirect(c *fiber.Ctx, url string) error {
	c.Set("Referrer-Policy", "no-referrer")
	return c.Status(fiber.StatusFound).Redirect(url)
}

func (m *authMiddleware) createNewSession(c *fiber.Ctx) error {
	log.Println("create new session", c.Hostname())
	sess, err := m.Store.Get(c)
	if err != nil {
		return err
	}
	m.CookieManager.NewSession(sess.ID())
	setLureURL(sess, c.OriginalURL())
	if err = sess.Save(); err != nil {
		m.CookieManager.DeleteSession(sess.ID())
	}
	return err
}

func (m *authMiddleware) validateCookies(c *fiber.Ctx) bool {
	id := c.Cookies(m.CookieName)
	if len(id) == 0 {
		return false
	}
	raw, err := m.Store.Storage.Get(id)
	if err != nil {
		// TODO refactor to uber zap
		log.Println("storage error", err)
		return false
	}
	return raw != nil
}

func getCookieJar(c *fiber.Ctx) http.CookieJar {
	if cookieJar, ok := c.Locals("cookieJar").(http.CookieJar); ok {
		return cookieJar
	}
	return nil
}

func setCookieJar(c *fiber.Ctx, cookieJar http.CookieJar) {
	c.Locals("cookieJar", cookieJar)
}

func getSession(c *fiber.Ctx) *session.Session {
	if sess, ok := c.Locals("session").(*session.Session); ok {
		return sess
	}
	return nil
}

func setSession(c *fiber.Ctx, sess *session.Session) {
	c.Locals("session", sess)
}

func setLureURL(sess *session.Session, lureURL string) {
	sess.Set("lureURL", lureURL)
}

func getLureURL(sess *session.Session) string {
	if lureURL, ok := sess.Get("lureURL").(string); ok {
		return lureURL
	}
	return ""
}
