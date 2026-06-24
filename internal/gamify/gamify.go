// Package gamify derives XP, levels, streaks and goal progress from sessions.
package gamify

import (
	"time"

	"github.com/xZhad/pomo/internal/model"
)

// XPFor is the XP awarded for one completed focus: base 10, +5 with a note,
// +the current streak (capped at 10).
func XPFor(hasNote bool, streak int) int {
	xp := 10
	if hasNote {
		xp += 5
	}
	if streak > 10 {
		streak = 10
	}
	return xp + streak
}

// TotalXP sums XP across completed sessions.
func TotalXP(sessions []model.Session) int {
	t := 0
	for _, s := range sessions {
		t += s.XP
	}
	return t
}

// Level returns the level for a total XP, plus XP into the current level and
// the span of that level. Level n starts at 50·n·(n-1) cumulative XP.
func Level(xp int) (level, into, span int) {
	n := 1
	for 50*(n+1)*n <= xp {
		n++
	}
	base := 50 * n * (n - 1)
	next := 50 * (n + 1) * n
	return n, xp - base, next - base
}

func day(t time.Time) string { return t.Format("2006-01-02") }

// completedDays is the set of local dates with ≥1 completed focus.
func completedDays(sessions []model.Session) map[string]bool {
	d := map[string]bool{}
	for _, s := range sessions {
		if s.Completed {
			d[day(s.Started)] = true
		}
	}
	return d
}

// Streak is the run of consecutive days with a completed focus, ending today
// (or yesterday if nothing yet today).
func Streak(sessions []model.Session, now time.Time) int {
	days := completedDays(sessions)
	cur := now
	if !days[day(cur)] {
		cur = cur.AddDate(0, 0, -1)
		if !days[day(cur)] {
			return 0
		}
	}
	streak := 0
	for days[day(cur)] {
		streak++
		cur = cur.AddDate(0, 0, -1)
	}
	return streak
}

// CompletedToday counts completed focus sessions started today (local).
func CompletedToday(sessions []model.Session, now time.Time) int {
	today := day(now)
	n := 0
	for _, s := range sessions {
		if s.Completed && day(s.Started) == today {
			n++
		}
	}
	return n
}
