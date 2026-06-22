package store

import (
	"os"
	"path/filepath"

	"github.com/xZhad/jsonldb"
	"github.com/xZhad/pomo/internal/model"
)

// Dir returns the pomo data directory (POMO_DIR or ~/.pomo).
func Dir() string {
	if d := os.Getenv("POMO_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pomo"
	}
	return filepath.Join(home, ".pomo")
}

type Store struct{ dir string }

func Open() (*Store, error) {
	d := Dir()
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: d}, nil
}

func (s *Store) SessionsPath() string { return filepath.Join(s.dir, "sessions.jsonl") }

func (s *Store) open() (*jsonldb.Collection, error) { return jsonldb.Open(s.SessionsPath()) }

func (s *Store) AppendSession(sess model.Session) error {
	c, err := s.open()
	if err != nil {
		return err
	}
	defer c.Close()
	return jsonldb.Typed[model.Session](c).Append(sess)
}

func (s *Store) AllSessions() ([]model.Session, error) {
	c, err := s.open()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	return jsonldb.Typed[model.Session](c).All()
}

func (s *Store) UpdateSession(id string, mut func(model.Session) model.Session) (int, error) {
	c, err := s.open()
	if err != nil {
		return 0, err
	}
	defer c.Close()
	return jsonldb.Typed[model.Session](c).Update(jsonldb.Eq("id", id), mut)
}

func (s *Store) DeleteSession(id string) (int, error) {
	c, err := s.open()
	if err != nil {
		return 0, err
	}
	defer c.Close()
	return jsonldb.Typed[model.Session](c).DeleteWhere(jsonldb.Eq("id", id))
}
