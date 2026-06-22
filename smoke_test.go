package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xZhad/jsonldb"
)

func TestJsonldbResolvable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(p, []byte("{\"id\":\"a\"}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := jsonldb.Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()
	if c.Count() != 1 {
		t.Fatalf("Count = %d, want 1", c.Count())
	}
}
