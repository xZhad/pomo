package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	WorkMin      int  `json:"work"`
	BreakMin     int  `json:"break"`       // short break
	LongBreakMin int  `json:"long_break"`  // long break (after a full cycle)
	CycleLength  int  `json:"cycle"`       // focuses per cycle before a long break
	Goal         int  `json:"goal"`        // daily focus goal
	Notify       bool `json:"notify"`
}

func DefaultConfig() Config {
	return Config{WorkMin: 25, BreakMin: 5, LongBreakMin: 15, CycleLength: 4, Goal: 4, Notify: true}
}

func (s *Store) configPath() string { return filepath.Join(s.dir, "config.json") }

// LoadConfig returns defaults overlaid with any values present in config.json.
func (s *Store) LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	b, err := os.ReadFile(s.configPath())
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (s *Store) SaveConfig(c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.configPath(), b)
}

// atomicWrite writes b to path via a temp file + rename.
func atomicWrite(path string, b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pomo-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
