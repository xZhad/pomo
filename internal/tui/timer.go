package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
	"github.com/xZhad/pomo/internal/gamify"
	"github.com/xZhad/pomo/internal/model"
	"github.com/xZhad/pomo/internal/report"
	"github.com/xZhad/pomo/internal/session"
)

type mode int

const (
	modeTimer mode = iota
	modeTopic
	modeNote
	modeHistory
	modeStats   // gamified dashboard
	modeAdvance // between phases: focus→break or break→focus
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type Model struct {
	svc         *session.Service
	bar         progress.Model
	input       textinput.Model
	spring      harmonica.Spring
	pulse       float64
	pulseVel    float64
	pulseTarget float64
	mode        mode
	status      session.Status
	notified    bool
	w, h        int
	history     []model.Session
	histCursor  int
	frame       int    // animation frame (logo shimmer)
	advanceKind string // "break" (after focus) | "focus" (after break)
	advanceLong bool   // next break is a long break
}

// cycleLength returns the configured focuses-per-cycle (default 4).
func (m *Model) cycleLength() int {
	cfg, err := m.svc.Store.LoadConfig()
	if err != nil || cfg.CycleLength <= 0 {
		return 4
	}
	return cfg.CycleLength
}

func New(svc *session.Service) *Model {
	ti := textinput.New()
	ti.Placeholder = "what are you working on?"
	ti.Focus() // focus immediately so tests can type without calling Init
	bar := progress.New(progress.WithDefaultBlend(), progress.WithWidth(40))
	m := &Model{
		svc:         svc,
		bar:         bar,
		input:       ti,
		spring:      harmonica.NewSpring(harmonica.FPS(10), 6.0, 0.4),
		pulseTarget: 1,
	}
	m.refresh()
	if !m.status.Active {
		m.mode = modeTopic
	}
	return m
}

func (m *Model) refresh() { m.status, _ = m.svc.Status() }

func (m *Model) Init() tea.Cmd { return tea.Batch(tick(), m.input.Focus()) }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.bar.SetWidth(min(50, msg.Width-10))
		return m, nil
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd
	case tickMsg:
		return m, m.onTick()
	case tea.KeyPressMsg:
		return m.onKey(msg)
	}
	return m, nil
}

func (m *Model) onTick() tea.Cmd {
	m.refresh()
	m.frame++
	var cmds []tea.Cmd
	if m.status.Active {
		dur := time.Duration(m.status.Session.Duration) * time.Second
		frac := 0.0
		if dur > 0 {
			frac = 1 - m.status.Remaining.Seconds()/dur.Seconds()
		}
		if frac < 0 {
			frac = 0
		} else if frac > 1 {
			frac = 1
		}
		cmds = append(cmds, m.bar.SetPercent(frac))
		if m.status.Remaining <= 0 && !m.notified && m.mode == modeTimer {
			m.notified = true
			if m.status.Phase == "focus" {
				_ = m.svc.Notifier.Notify("pomo", "Focus complete — break time 🍅")
				_, _ = m.svc.Done()
				n, _ := m.svc.CompletedFocusToday()
				cyc := m.cycleLength()
				m.advanceLong = cyc > 0 && n%cyc == 0
				m.advanceKind = "break"
			} else {
				_ = m.svc.Notifier.Notify("pomo", "Break over — back to focus 🍅")
				_ = m.svc.EndBreak()
				m.advanceKind = "focus"
			}
			m.mode = modeAdvance
			m.refresh()
		}
	}
	// breathing pulse
	m.pulse, m.pulseVel = m.spring.Update(m.pulse, m.pulseVel, m.pulseTarget)
	if (m.pulseTarget == 1 && m.pulse > 0.98) || (m.pulseTarget == 0 && m.pulse < 0.02) {
		m.pulseTarget = 1 - m.pulseTarget
	}
	cmds = append(cmds, tick())
	return tea.Batch(cmds...)
}

func (m *Model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeTopic:
		switch msg.String() {
		case "enter":
			topic := m.input.Value()
			if topic != "" {
				if _, err := m.svc.Start(session.StartOpts{Topic: topic}); err == nil {
					m.input.SetValue("")
					m.notified = false
					m.mode = modeTimer
					m.refresh()
				}
			}
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	case modeNote:
		switch msg.String() {
		case "enter":
			if v := m.input.Value(); v != "" {
				_ = m.svc.Note(v)
			}
			m.input.SetValue("")
			m.mode = modeTimer
			return m, nil
		case "esc":
			m.input.SetValue("")
			m.mode = modeTimer
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	case modeHistory:
		switch msg.String() {
		case "tab", "esc":
			m.mode = modeTimer
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.histCursor < len(m.history)-1 {
				m.histCursor++
			}
		case "k", "up":
			if m.histCursor > 0 {
				m.histCursor--
			}
		}
		return m, nil
	case modeStats:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.history, _ = report.Log(m.svc.Store, "")
			m.histCursor = 0
			m.mode = modeHistory
		case "esc":
			m.mode = modeTimer
		}
		return m, nil
	case modeAdvance:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter", " ":
			if m.advanceKind == "break" {
				_, _ = m.svc.StartBreak(m.advanceLong)
				m.notified = false
				m.mode = modeTimer
				m.refresh()
			} else {
				m.toTopic()
			}
		case "s": // skip the break / go straight to a new focus
			m.toTopic()
		}
		return m, nil
	default: // modeTimer
		focus := m.status.Phase == "focus"
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.mode = modeStats
			return m, nil
		case "p", " ":
			if m.status.Paused {
				_ = m.svc.Resume()
			} else {
				_ = m.svc.Pause()
			}
			m.refresh()
		case "n":
			if focus { // notes only attach to a focus session
				m.input.SetValue("")
				m.input.Placeholder = "note…"
				m.mode = modeNote
				return m, m.input.Focus()
			}
		case "e":
			_ = m.svc.Extend(5 * time.Minute)
			m.refresh()
		case "d": // finish this phase
			if focus {
				_, _ = m.svc.Done()
				n, _ := m.svc.CompletedFocusToday()
				cyc := m.cycleLength()
				m.advanceLong = cyc > 0 && n%cyc == 0
				m.advanceKind = "break"
				m.notified = true
				m.mode = modeAdvance
				m.refresh()
			} else { // end break early → next focus
				_ = m.svc.EndBreak()
				m.toTopic()
			}
		case "s": // stop/abort
			if focus {
				_, _ = m.svc.Stop()
			} else {
				_ = m.svc.EndBreak()
			}
			m.toTopic()
		}
		return m, nil
	}
}

func (m *Model) toTopic() {
	m.input.SetValue("")
	m.input.Placeholder = "what are you working on?"
	m.mode = modeTopic
	m.notified = false
	m.refresh()
}

func (m *Model) View() tea.View {
	if m.mode == modeStats {
		v := tea.NewView(m.renderStats(m.w, m.h))
		v.AltScreen = true
		return v
	}
	phase := m.status.Phase
	if phase == "" {
		phase = "focus"
	}
	var inner string
	border := cViolet
	switch m.mode {
	case modeTopic:
		inner = m.viewTopic()
	case modeNote:
		inner = m.viewNote()
	case modeHistory:
		inner = m.viewHistory()
	case modeAdvance:
		inner = m.viewAdvance()
		if m.advanceKind == "break" {
			next := "short"
			if m.advanceLong {
				next = "long"
			}
			border = phaseColor(next)
		}
	default:
		inner = m.viewTimer(phase)
		border = phaseColor(phase)
	}
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Padding(1, 4).Render(inner)
	out := card
	if m.w > 0 && m.h > 0 {
		out = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, card)
	}
	v := tea.NewView(out)
	v.AltScreen = true
	return v
}

func (m *Model) viewTimer(phase string) string {
	s := m.status
	mm := int(s.Remaining.Minutes())
	ss := int(s.Remaining.Seconds()) % 60
	if mm < 0 {
		mm, ss = 0, 0
	}
	clock := bigTime(fmt.Sprintf("%02d:%02d", mm, ss), phaseStops(phase)...)
	label := lipgloss.NewStyle().Foreground(phaseColor(phase)).Bold(true).Render(phaseLabel(phase))
	subject := styleTopic.Render(s.Session.Topic)
	if phase != "focus" {
		subject = styleMuted.Render("relax & recharge")
	}
	rows := []string{
		"🍅 " + gradientText("pomo", m.frame),
		m.statsLine(),
		"",
		clock,
		"",
		label + styleMuted.Render("  ·  ") + subject,
		m.bar.View(),
		m.cycleDots(),
	}
	if s.Paused {
		rows = append(rows, styleWarn.Render("⏸ paused"))
	}
	hints := keyHint("p", "pause") + keyHint("e", "+5m") + keyHint("d", "done") +
		keyHint("s", "stop") + keyHint("tab", "stats") + keyHint("q", "quit")
	if phase == "focus" {
		hints = keyHint("p", "pause") + keyHint("n", "note") + keyHint("e", "+5m") +
			keyHint("d", "done") + keyHint("s", "stop") + keyHint("tab", "stats") + keyHint("q", "quit")
	}
	rows = append(rows, "", hints)
	return lipgloss.JoinVertical(lipgloss.Center, rows...)
}

// statsLine renders the gamification status: streak flame, daily-goal ring,
// level + XP bar — derived from the session log.
func (m *Model) statsLine() string {
	all, _ := m.svc.Store.AllSessions()
	now := m.svc.Now()
	streak := gamify.Streak(all, now)
	done := gamify.CompletedToday(all, now)
	cfg, _ := m.svc.Store.LoadConfig()
	goal := cfg.Goal
	if goal <= 0 {
		goal = 4
	}
	lvl, into, span := gamify.Level(gamify.TotalXP(all))
	flame := styleMuted.Render("· no streak ·")
	if streak > 0 {
		flame = styleWarn.Render(fmt.Sprintf("🔥 %d", streak))
	}
	goalStr := goalRing(done, goal) + styleMuted.Render(fmt.Sprintf(" %d/%d", done, goal))
	lvlStr := styleKey.Render(fmt.Sprintf("lvl %d ", lvl)) + miniBar(into, span, 8)
	return flame + styleMuted.Render("   ") + goalStr + styleMuted.Render("   ") + lvlStr
}

// cycleDots shows progress through the current pomodoro cycle.
func (m *Model) cycleDots() string {
	cyc := m.cycleLength()
	n, _ := m.svc.CompletedFocusToday()
	pos := 0
	if cyc > 0 {
		pos = n % cyc
	}
	var b strings.Builder
	b.WriteString(styleMuted.Render("cycle  "))
	for i := 0; i < cyc; i++ {
		if i < pos {
			b.WriteString(lipgloss.NewStyle().Foreground(cMagenta).Render("● "))
		} else {
			b.WriteString(styleMuted.Render("○ "))
		}
	}
	return b.String()
}

func (m *Model) viewAdvance() string {
	var head, sub string
	if m.advanceKind == "break" {
		kind := "short break"
		col := cCyan
		if m.advanceLong {
			kind, col = "long break", cGreen
		}
		head = styleOK.Render("✓  FOCUS COMPLETE")
		sub = styleMuted.Render("↵ start ") +
			lipgloss.NewStyle().Foreground(col).Bold(true).Render(kind) +
			styleMuted.Render("   ·   s skip   ·   q quit")
	} else {
		head = lipgloss.NewStyle().Foreground(cCyan).Bold(true).Render("☕  BREAK OVER")
		sub = styleMuted.Render("↵ next focus   ·   q quit")
	}
	return lipgloss.JoinVertical(lipgloss.Center,
		"🍅 "+gradientText("pomo", m.frame), m.statsLine(), "", head, "", m.cycleDots(), "", sub)
}

func (m *Model) viewTopic() string {
	return lipgloss.JoinVertical(lipgloss.Center,
		"🍅 "+gradientText("pomo", m.frame), "",
		styleMuted.Render("what are you working on?"),
		m.input.View(), "",
		styleMuted.Render("enter start · ctrl+c quit"))
}

func (m *Model) viewNote() string {
	return lipgloss.JoinVertical(lipgloss.Center,
		"🍅 "+gradientText("pomo", m.frame)+styleMuted.Render(" · note"), "",
		m.input.View(), "",
		styleMuted.Render("enter save · esc cancel"))
}

func (m *Model) viewHistory() string {
	var b strings.Builder
	b.WriteString("🍅 " + gradientText("pomo", m.frame) + styleMuted.Render(" · history") + "\n\n")
	if len(m.history) == 0 {
		b.WriteString(styleMuted.Render("(no sessions yet)"))
	}
	for i, s := range m.history {
		cur := "  "
		st := styleText
		if i == m.histCursor {
			cur = lipgloss.NewStyle().Foreground(cMagenta).Bold(true).Render("▌ ")
			st = styleTopic
		}
		mark := styleOK.Render("✓")
		if !s.Completed {
			mark = lipgloss.NewStyle().Foreground(cRed).Render("✗")
		}
		b.WriteString(fmt.Sprintf("%s%s %s  %s  %s\n",
			cur, mark, styleMuted.Render(s.Started.Format("01-02 15:04")),
			st.Render(fmt.Sprintf("%-20s", trunc(s.Topic, 20))), styleKey.Render(fmt.Sprintf("%dm", s.Duration/60))))
	}
	b.WriteString("\n" + keyHint("tab", "back") + keyHint("j/k", "move") + keyHint("q", "quit"))
	return b.String()
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
