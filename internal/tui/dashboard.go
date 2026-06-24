package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/xZhad/pomo/internal/gamify"
)

func (m *Model) stats() gamify.Stats {
	all, _ := m.svc.Store.AllSessions()
	cfg, _ := m.svc.Store.LoadConfig()
	return gamify.Compute(all, m.svc.Now(), cfg.Goal)
}

func sectionTitle(s string) string {
	return lipgloss.NewStyle().Foreground(cCyan).Bold(true).Render(s)
}

func (m *Model) renderStats(w, h int) string {
	s := m.stats()
	colW := (w - 6) / 2
	if colW < 24 {
		colW = 24
	}

	header := "🍅 " + gradientText("pomo", m.frame) + styleMuted.Render(" · stats")
	lvl := styleKey.Render(fmt.Sprintf("lvl %d ", s.Level)) + miniBar(s.IntoLevel, s.LevelSpan, 14) +
		styleMuted.Render(fmt.Sprintf(" %d/%d XP", s.IntoLevel, s.LevelSpan))
	flame := styleMuted.Render("· no streak ·")
	if s.Streak > 0 {
		flame = styleWarn.Render(fmt.Sprintf("🔥 %d", s.Streak))
	}
	totals := fmt.Sprintf("%dh%02dm · %d focuses", s.TotalMinutes/60, s.TotalMinutes%60, s.TotalSessions)
	summary := flame + styleMuted.Render("   ") + goalRing(s.Today, s.Goal) +
		styleMuted.Render(fmt.Sprintf(" %d/%d today   ", s.Today, s.Goal)) + styleMuted.Render(totals)

	left := lipgloss.JoinVertical(lipgloss.Left,
		sectionTitle("this week"), weekChart(s, colW),
		"", sectionTitle("top topics"), topicBars(s.TopTopics, colW))
	right := lipgloss.JoinVertical(lipgloss.Left,
		sectionTitle("focus calendar"), calendarHeat(s.Days, m.svc.Now(), colW),
		"", sectionTitle("badges"), badgeGrid(s.Badges))
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(colW+2).Render(left), right)

	best := styleMuted.Render("— no sessions yet —")
	if s.BestCount > 0 {
		best = styleMuted.Render("★ best day ") + styleKey.Render(s.BestDay) +
			styleMuted.Render(fmt.Sprintf(" (%d)", s.BestCount))
	}
	body := lipgloss.JoinVertical(lipgloss.Left,
		header, lvl, summary, gradientRule(min(w-2, 72)), cols, "", best, "",
		keyHint("tab", "history")+keyHint("esc", "timer")+keyHint("q", "quit"))

	card := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cViolet).
		Padding(1, 2).Render(body)
	if m.w > 0 && m.h > 0 {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, card)
	}
	return card
}

// weekChart is an ntcharts bar chart of the last 7 days' focus counts.
func weekChart(s gamify.Stats, w int) string {
	if w < 14 {
		w = 14
	}
	ramp := lipgloss.Blend1D(7, cViolet, cMagenta, cCyan)
	bc := barchart.New(w, 7)
	data := make([]barchart.BarData, 7)
	for i := 0; i < 7; i++ {
		data[i] = barchart.BarData{
			Label:  s.WeekLabels[i],
			Values: []barchart.BarValue{{Name: "d", Value: float64(s.Week[i]), Style: lipgloss.NewStyle().Foreground(ramp[i])}},
		}
	}
	bc.PushAll(data)
	bc.Draw()
	return bc.View()
}

// calendarHeat is a GitHub-style by-day focus grid (7 weekday rows × weeks).
func calendarHeat(days map[string]int, now time.Time, w int) string {
	weeks := (w - 5) / 2
	if weeks < 4 {
		weeks = 4
	}
	if weeks > 20 {
		weeks = 20
	}
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := end.AddDate(0, 0, -int(end.Weekday())-(weeks-1)*7) // back to Sunday of the first week
	maxC := 1
	for _, c := range days {
		if c > maxC {
			maxC = c
		}
	}
	ramp := lipgloss.Blend1D(8, cBg, cIdle, cViolet, cMagenta)
	labels := []string{"  ", "Mo", "  ", "We", "  ", "Fr", "  "}
	var b strings.Builder
	for wd := 0; wd < 7; wd++ {
		b.WriteString(styleMuted.Render(labels[wd]) + " ")
		for wk := 0; wk < weeks; wk++ {
			d := start.AddDate(0, 0, wk*7+wd)
			if d.After(end) {
				b.WriteString("  ")
				continue
			}
			n := days[d.Format("2006-01-02")]
			b.WriteString(lipgloss.NewStyle().Foreground(ramp[n*(len(ramp)-1)/maxC]).Render("██"))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func topicBars(tops []gamify.TopicCount, w int) string {
	if len(tops) == 0 {
		return styleMuted.Render("— none yet —")
	}
	maxC := 1
	for _, t := range tops {
		if t.Count > maxC {
			maxC = t.Count
		}
	}
	barW := w - 18
	if barW < 4 {
		barW = 4
	}
	ramp := []color.Color{cMagenta, cViolet, cCyan, cYellow, cGreen, cOrange}
	var b strings.Builder
	for i, t := range tops {
		fill := t.Count * barW / maxC
		if fill < 1 {
			fill = 1
		}
		bar := lipgloss.NewStyle().Foreground(ramp[i%len(ramp)]).Render(strings.Repeat("▰", fill))
		b.WriteString(fmt.Sprintf("%s %s %s\n",
			styleText.Render(fmt.Sprintf("%-10s", trunc(t.Topic, 10))), bar, styleKey.Render(fmt.Sprintf("%d", t.Count))))
	}
	return strings.TrimRight(b.String(), "\n")
}

func badgeGrid(badges []gamify.Badge) string {
	earned := 0
	var b strings.Builder
	for i, bd := range badges {
		if bd.Earned {
			earned++
			b.WriteString(bd.Icon + " ")
		} else {
			b.WriteString(styleMuted.Render("· "))
		}
		if (i+1)%5 == 0 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n" + styleMuted.Render(fmt.Sprintf("%d / %d unlocked", earned, len(badges))))
	return b.String()
}
