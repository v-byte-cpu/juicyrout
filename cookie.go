//go:generate mockgen -package main -destination=cookie_mock_test.go -source=cookie.go
package main

import (
	"net/http"
	"net/http/cookiejar"
	"sync"

	"github.com/gofiber/fiber/v2"
)

type CookieManager interface {
	NewSession(sessionID string) http.CookieJar
	Get(sessionID string) http.CookieJar
	Delete(sessionID string)
}

func NewCookieManager() CookieManager {
	return &cookieManager{
		sessions: make(map[string]http.CookieJar),
	}
}

type cookieManager struct {
	// mu locks the remaining fields.
	mu sync.RWMutex
	// sessions maps sessionId to CookieJar
	sessions map[string]http.CookieJar
}

func (c *cookieManager) NewSession(sessionID string) http.CookieJar {
	cookie, _ := cookiejar.New(nil)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessions[sessionID] = cookie
	return cookie
}

func (c *cookieManager) Get(sessionID string) http.CookieJar {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessions[sessionID]
}

func (c *cookieManager) Delete(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.sessions, sessionID)
}

func NewSessionStorage(delegate fiber.Storage, cookieManager CookieManager) fiber.Storage {
	return &sessionStorage{delegate, cookieManager}
}

type sessionStorage struct {
	fiber.Storage
	cookieManager CookieManager
}

func (s *sessionStorage) Delete(key string) error {
	s.cookieManager.Delete(key)
	return s.Storage.Delete(key)
}
