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
	frame       int // animation frame (logo shimmer)
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
		if m.status.Remaining <= 0 && !m.notified {
			_ = m.svc.Notifier.Notify("pomo", "Time's up — break 🍅")
			m.notified = true
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
	default: // modeTimer
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.history, _ = report.Log(m.svc.Store, "")
			m.histCursor = 0
			m.mode = modeHistory
			return m, nil
		case "p", " ":
			if m.status.Paused {
				_ = m.svc.Resume()
			} else {
				_ = m.svc.Pause()
			}
			m.refresh()
		case "n":
			m.input.SetValue("")
			m.input.Placeholder = "note…"
			m.mode = modeNote
			return m, m.input.Focus()
		case "d":
			_, _ = m.svc.Done()
			m.toTopic()
		case "s":
			_, _ = m.svc.Stop()
			m.toTopic()
		case "e":
			_ = m.svc.Extend(5 * time.Minute)
			m.refresh()
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
	phase := "focus" // batch 2 wires real work/break phases
	var inner string
	border := cViolet
	switch m.mode {
	case modeTopic:
		inner = m.viewTopic()
	case modeNote:
		inner = m.viewNote()
	case modeHistory:
		inner = m.viewHistory()
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
	rows := []string{
		"🍅 " + gradientText("pomo", m.frame),
		"",
		clock,
		"",
		label + styleMuted.Render("  ·  ") + styleTopic.Render(s.Session.Topic),
		m.bar.View(),
	}
	if s.Paused {
		rows = append(rows, styleWarn.Render("⏸ paused"))
	} else if s.Remaining <= 0 {
		rows = append(rows, styleOK.Render("✓ done — break time"))
	}
	rows = append(rows, "",
		keyHint("p", "pause")+keyHint("n", "note")+keyHint("e", "+5m")+
			keyHint("d", "done")+keyHint("s", "stop")+keyHint("tab", "stats")+keyHint("q", "quit"))
	return lipgloss.JoinVertical(lipgloss.Center, rows...)
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
