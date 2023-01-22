//go:generate mockgen -package main -destination=session_mock_test.go -source=session.go
package main

import (
	"log"
	"net/http"
	"net/http/cookiejar"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type SessionManager interface {
	NewSession(c *fiber.Ctx, lureURL string) (*proxySession, error)
	GetSession(c *fiber.Ctx) (*proxySession, error)
	GetOrCreateSession(c *fiber.Ctx, lureURL string) (*proxySession, error)
	DeleteSession(sessionID string)
}

func NewSessionManager(store *session.Store, sessionCookieName string) SessionManager {
	return &sessionManager{
		sessions:   make(map[string]*proxySession),
		store:      store,
		cookieName: sessionCookieName,
	}
}

type sessionManager struct {
	// mu locks the remaining fields.
	mu sync.RWMutex
	// sessions maps sessionId to session object
	sessions map[string]*proxySession
	// Store interface to store the session data
	store *session.Store
	// Name of authentication cookie
	cookieName string
}

func (m *sessionManager) NewSession(c *fiber.Ctx, lureURL string) (*proxySession, error) {
	log.Println("create new session", c.Hostname(), lureURL)

	sess, err := m.store.Get(c)
	if err != nil {
		return nil, err
	}
	cookie, _ := cookiejar.New(nil)
	newSess := &proxySession{cookieJar: cookie, lureURL: lureURL}
	m.mu.Lock()
	m.sessions[sess.ID()] = newSess
	m.mu.Unlock()

	resultSess := *newSess
	resultSess.Session = sess
	return &resultSess, nil
}

func (m *sessionManager) GetSession(c *fiber.Ctx) (result *proxySession, err error) {
	id := c.Cookies(m.cookieName)
	if len(id) == 0 {
		return
	}
	m.mu.RLock()
	proxySess := m.sessions[id]
	m.mu.RUnlock()
	if proxySess == nil {
		return
	}
	sess, err := m.store.Get(c)
	if err != nil {
		return nil, err
	}
	if sess.ID() != id {
		return
	}
	resultSess := *proxySess
	resultSess.Session = sess
	return &resultSess, nil
}

func (m *sessionManager) GetOrCreateSession(c *fiber.Ctx, lureURL string) (*proxySession, error) {
	proxySess, err := m.GetSession(c)
	if err != nil || proxySess != nil {
		return proxySess, err
	}
	return m.NewSession(c, lureURL)
}

func (m *sessionManager) DeleteSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

type proxySession struct {
	*session.Session
	lureURL   string
	cookieJar http.CookieJar
}

func (s *proxySession) CookieJar() http.CookieJar {
	return s.cookieJar
}

func (s *proxySession) LureURL() string {
	return s.lureURL
}

type SessionDeleter interface {
	DeleteSession(sessionID string)
}

func NewSessionStorage(delegate fiber.Storage, deleteCallbacks ...SessionDeleter) SessionStorage {
	return &sessionStorage{delegate, deleteCallbacks}
}

type SessionStorage interface {
	fiber.Storage
	AddSessionDeleter(sd SessionDeleter)
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

func (s *sessionStorage) AddSessionDeleter(sd SessionDeleter) {
	s.deleteCallbacks = append(s.deleteCallbacks, sd)
}
