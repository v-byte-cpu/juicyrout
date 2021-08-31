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
	GetSession(sessionID string) http.CookieJar
	DeleteSession(sessionID string)
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

func (c *cookieManager) GetSession(sessionID string) http.CookieJar {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessions[sessionID]
}

func (c *cookieManager) DeleteSession(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.sessions, sessionID)
}

type SessionDeleter interface {
	DeleteSession(sessionID string)
}

func NewSessionStorage(delegate fiber.Storage, deleteCallbacks ...SessionDeleter) fiber.Storage {
	return &sessionStorage{delegate, deleteCallbacks}
}

type sessionStorage struct {
	fiber.Storage
	deleteCallbacks []SessionDeleter
}

func (s *sessionStorage) Delete(key string) error {
	for _, callback := range s.deleteCallbacks {
		callback.DeleteSession(key)
	}
	return s.Storage.Delete(key)
}
