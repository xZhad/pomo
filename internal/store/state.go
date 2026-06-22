package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	ID       string    `json:"id"`
	Started  time.Time `json:"started"`
	Deadline time.Time `json:"deadline"`
	Paused   bool      `json:"paused"`
	PausedAt time.Time `json:"paused_at,omitempty"`
}

func (s *Store) statePath() string { return filepath.Join(s.dir, "current.json") }

func (s *Store) LoadState() (State, bool, error) {
	var st State
	b, err := os.ReadFile(s.statePath())
	if os.IsNotExist(err) {
		return st, false, nil
	}
	if err != nil {
		return st, false, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, false, err
	}
	return st, true, nil
}

func (s *Store) SaveState(st State) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.statePath(), b)
}

func (s *Store) ClearState() error {
	err := os.Remove(s.statePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
