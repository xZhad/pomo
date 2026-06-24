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

// TopicCount is a topic with its completed-session count.
type TopicCount struct {
	Topic string
	Count int
}

// Badge is an achievement, earned or locked.
type Badge struct {
	Icon, Name, Desc string
	Earned           bool
}

// Stats is the full dashboard aggregate, computed once from the session log.
type Stats struct {
	Level, XP, IntoLevel, LevelSpan int
	Streak, Today, Goal             int
	TotalSessions, TotalMinutes     int
	Week                            [7]int    // counts for the last 7 days, ending today
	WeekLabels                      [7]string // weekday initials, aligned to Week
	Days                            map[string]int
	TopTopics                       []TopicCount
	BestDay                         string
	BestCount                       int
	Badges                          []Badge
}

// Compute derives the dashboard stats from completed sessions.
func Compute(sessions []model.Session, now time.Time, goal int) Stats {
	if goal <= 0 {
		goal = 4
	}
	var st Stats
	st.Goal = goal
	st.XP = TotalXP(sessions)
	st.Level, st.IntoLevel, st.LevelSpan = Level(st.XP)
	st.Streak = Streak(sessions, now)
	st.Today = CompletedToday(sessions, now)
	st.Days = map[string]int{}
	topics := map[string]int{}
	for _, s := range sessions {
		if !s.Completed {
			continue
		}
		st.TotalSessions++
		st.TotalMinutes += s.Duration / 60
		st.Days[day(s.Started)]++
		topics[s.Topic]++
	}
	// last 7 days
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, -(6 - i))
		st.Week[i] = st.Days[day(d)]
		st.WeekLabels[i] = d.Format("Mon")[:1]
	}
	// best day
	for d, c := range st.Days {
		if c > st.BestCount {
			st.BestCount, st.BestDay = c, d
		}
	}
	// top topics
	for t, c := range topics {
		st.TopTopics = append(st.TopTopics, TopicCount{t, c})
	}
	sortTopics(st.TopTopics)
	if len(st.TopTopics) > 6 {
		st.TopTopics = st.TopTopics[:6]
	}
	st.Badges = Badges(sessions, now, st)
	return st
}

func sortTopics(ts []TopicCount) {
	for i := 1; i < len(ts); i++ { // small n; simple insertion sort, count desc then name
		for j := i; j > 0 && (ts[j].Count > ts[j-1].Count ||
			(ts[j].Count == ts[j-1].Count && ts[j].Topic < ts[j-1].Topic)); j-- {
			ts[j], ts[j-1] = ts[j-1], ts[j]
		}
	}
}

// Badges computes achievement state from the session log.
func Badges(sessions []model.Session, now time.Time, st Stats) []Badge {
	var early, night, marathon bool
	topicMax := 0
	tc := map[string]int{}
	goalDays := 0
	perDay := map[string]int{}
	for _, s := range sessions {
		if !s.Completed {
			continue
		}
		h := s.Started.Hour()
		if h < 7 {
			early = true
		}
		if h >= 22 {
			night = true
		}
		if s.Duration >= 3000 {
			marathon = true
		}
		tc[s.Topic]++
		if tc[s.Topic] > topicMax {
			topicMax = tc[s.Topic]
		}
		perDay[day(s.Started)]++
	}
	for _, c := range perDay {
		if c >= st.Goal {
			goalDays++
		}
	}
	return []Badge{
		{"🍅", "First Tomato", "complete a focus", st.TotalSessions >= 1},
		{"💯", "Centurion", "100 focuses", st.TotalSessions >= 100},
		{"🔥", "Week Warrior", "7-day streak", st.Streak >= 7},
		{"⚡", "Unstoppable", "30-day streak", st.Streak >= 30},
		{"🌅", "Early Bird", "focus before 7am", early},
		{"🦉", "Night Owl", "focus after 10pm", night},
		{"🏃", "Marathon", "a 50m+ focus", marathon},
		{"🎯", "Topic Master", "20 in one topic", topicMax >= 20},
		{"🏆", "Goal Crusher", "hit goal 10 days", goalDays >= 10},
	}
}
