package gamify

import (
	"testing"
	"time"

	"github.com/xZhad/pomo/internal/model"
)

func TestXPFor(t *testing.T) {
	if XPFor(false, 0) != 10 {
		t.Errorf("base = %d, want 10", XPFor(false, 0))
	}
	if XPFor(true, 3) != 18 {
		t.Errorf("note+streak = %d, want 18", XPFor(true, 3))
	}
	if XPFor(true, 50) != 25 {
		t.Errorf("streak capped = %d, want 25", XPFor(true, 50))
	}
}

func TestLevel(t *testing.T) {
	for _, tc := range []struct{ xp, lvl, into, span int }{
		{0, 1, 0, 100}, {50, 1, 50, 100}, {100, 2, 0, 200}, {300, 3, 0, 300},
	} {
		l, i, s := Level(tc.xp)
		if l != tc.lvl || i != tc.into || s != tc.span {
			t.Errorf("Level(%d) = (%d,%d,%d), want (%d,%d,%d)", tc.xp, l, i, s, tc.lvl, tc.into, tc.span)
		}
	}
}

func TestStreak(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	mk := func(daysAgo int) model.Session {
		return model.Session{Completed: true, Started: now.AddDate(0, 0, -daysAgo)}
	}
	// today, yesterday, 2 days ago → streak 3; gap at 3 stops it
	ss := []model.Session{mk(0), mk(1), mk(2), mk(4)}
	if got := Streak(ss, now); got != 3 {
		t.Errorf("streak = %d, want 3", got)
	}
	// nothing today but yesterday → still counts from yesterday
	ss2 := []model.Session{mk(1), mk(2)}
	if got := Streak(ss2, now); got != 2 {
		t.Errorf("streak (from yesterday) = %d, want 2", got)
	}
	// nothing recent → 0
	if got := Streak([]model.Session{mk(5)}, now); got != 0 {
		t.Errorf("streak = %d, want 0", got)
	}
}
