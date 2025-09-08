package logstore

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"orderpulse-api/internal/models"
)

type Store interface {
	Append(models.OrderEvent) error
	ReplaySince(time.Time, func(models.OrderEvent) bool) error
}

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err == nil {
		_ = f.Close()
	}
	return &FileStore{path: path}, err
}

func (s *FileStore) Append(ev models.OrderEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(ev)
	_, err = f.Write(append(b, '\n'))
	return err
}

func (s *FileStore) ReplaySince(since time.Time, yield func(models.OrderEvent) bool) error {
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ev models.OrderEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.TS.After(since) {
			if cont := yield(ev); !cont {
				break
			}
		}
	}
	return sc.Err()
}

var ErrStop = errors.New("stop")
