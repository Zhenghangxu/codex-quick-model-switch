package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"codex-quick-model-switch/internal/config"
)

type ActiveState struct {
	Active config.Switch `json:"active"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (ActiveState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return ActiveState{}, nil
	}
	if err != nil {
		return ActiveState{}, err
	}
	var st ActiveState
	if err := json.Unmarshal(data, &st); err != nil {
		return ActiveState{}, err
	}
	return st, nil
}

func (s *Store) Save(st ActiveState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.path, data, 0o600)
}
