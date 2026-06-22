package daemon

import (
	"testing"
	"time"

	"github.com/xZhad/pomo/internal/notify"
	"github.com/xZhad/pomo/internal/store"
)

func TestWatchFiresAtDeadline(t *testing.T) {
	t.Setenv("POMO_DIR", t.TempDir())
	s, _ := store.Open()
	base := time.Date(2026, 6, 8, 18, 30, 0, 0, time.UTC)
	s.SaveState(store.State{ID: "x", Started: base, Deadline: base.Add(25 * time.Minute)})

	rec := &notify.Recorder{}
	calls := 0
	now := func() time.Time {
		calls++
		if calls > 3 {
			return base.Add(26 * time.Minute) // past deadline after a few ticks
		}
		return base.Add(time.Duration(calls) * time.Minute)
	}
	if err := Watch(s, rec, "x", now, time.Millisecond, 50); err != nil {
		t.Fatal(err)
	}
	if len(rec.Calls) != 1 {
		t.Errorf("notify calls = %v, want 1", rec.Calls)
	}
}

func TestWatchStaleSessionExits(t *testing.T) {
	t.Setenv("POMO_DIR", t.TempDir())
	s, _ := store.Open()
	base := time.Date(2026, 6, 8, 18, 30, 0, 0, time.UTC)
	s.SaveState(store.State{ID: "y", Deadline: base.Add(25 * time.Minute)})
	rec := &notify.Recorder{}
	// watching for a DIFFERENT id -> stale, exits without notifying
	if err := Watch(s, rec, "x", func() time.Time { return base.Add(time.Hour) }, time.Millisecond, 10); err != nil {
		t.Fatal(err)
	}
	if len(rec.Calls) != 0 {
		t.Errorf("stale watch should not notify, got %v", rec.Calls)
	}
}
