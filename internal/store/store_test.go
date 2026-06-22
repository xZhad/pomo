package store

import (
	"testing"

	"github.com/xZhad/pomo/internal/model"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("POMO_DIR", t.TempDir())
	s, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSessionsCRUD(t *testing.T) {
	s := newStore(t)
	if err := s.AppendSession(model.Session{ID: "a", Topic: "ml", Duration: 1500}); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendSession(model.Session{ID: "b", Topic: "go", Duration: 900}); err != nil {
		t.Fatal(err)
	}
	all, err := s.AllSessions()
	if err != nil || len(all) != 2 {
		t.Fatalf("AllSessions len=%d err=%v", len(all), err)
	}
	n, err := s.UpdateSession("a", func(x model.Session) model.Session { x.Completed = true; return x })
	if err != nil || n != 1 {
		t.Fatalf("Update n=%d err=%v", n, err)
	}
	all, _ = s.AllSessions()
	var found bool
	for _, x := range all {
		if x.ID == "a" {
			found = x.Completed
		}
	}
	if !found {
		t.Error("update not persisted")
	}
	if n, _ := s.DeleteSession("b"); n != 1 {
		t.Errorf("delete n=%d, want 1", n)
	}
}
