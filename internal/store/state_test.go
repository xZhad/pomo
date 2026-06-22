package store

import (
	"testing"
	"time"
)

func TestStateLifecycle(t *testing.T) {
	s := newStore(t)
	if _, ok, err := s.LoadState(); err != nil || ok {
		t.Fatalf("expected no active state, ok=%v err=%v", ok, err)
	}
	st := State{ID: "a", Started: time.Now().UTC().Truncate(time.Second), Deadline: time.Now().UTC().Add(25 * time.Minute).Truncate(time.Second)}
	if err := s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.LoadState()
	if err != nil || !ok {
		t.Fatalf("LoadState ok=%v err=%v", ok, err)
	}
	if got.ID != "a" || !got.Deadline.Equal(st.Deadline) {
		t.Errorf("state mismatch: %+v", got)
	}
	if err := s.ClearState(); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.LoadState(); ok {
		t.Error("state not cleared")
	}
}
