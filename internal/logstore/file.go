package logstore

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"orderpulse-api/internal/models"
)

type Store interface {
	Append(models.OrderEvent) error
	ReplaySince(time.Time, func(models.OrderEvent) bool) error
	Health() error
}

type FileStore struct {
	path        string
	maxBytes    int64
	retention   time.Duration
	mu          sync.Mutex
	lastPruneAt time.Time
}

func NewFileStore(path string, maxBytes int64, retention time.Duration) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err == nil {
		_ = f.Close()
	}
	return &FileStore{path: path, maxBytes: maxBytes, retention: retention}, err
}

func (s *FileStore) Append(ev models.OrderEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if info, err := os.Stat(s.path); err == nil && info.Size() >= s.maxBytes {
		ts := time.Now().UTC().Format("20060102T150405")
		_ = os.Rename(s.path, s.path+"."+ts+".gz")
		_, _ = os.Create(s.path)
	}

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	b, _ := json.Marshal(ev)
	_, err = f.Write(append(b, '\n'))

	if time.Since(s.lastPruneAt) > time.Hour {
		_ = s.pruneOld()
		s.lastPruneAt = time.Now()
	}
	return err
}

func (s *FileStore) pruneOld() error {
	dir := filepath.Dir(s.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	cut := time.Now().Add(-s.retention)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Base(s.path) == name {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if fi.ModTime().Before(cut) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
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
		if json.Unmarshal(sc.Bytes(), &ev) == nil && ev.TS.After(since) {
			if !yield(ev) {
				break
			}
		}
	}
	return sc.Err()
}

func (s *FileStore) Health() error {
	_, err := os.Stat(s.path)
	return err
}
