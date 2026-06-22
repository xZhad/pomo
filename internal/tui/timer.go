package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
	"github.com/xZhad/pomo/internal/model"
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
	default: // modeTimer
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
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
	var body string
	switch m.mode {
	case modeTopic:
		body = fmt.Sprintf("🍅 pomo\n\nstart a session:\n%s\n\n(enter to start · ctrl+c quit)", m.input.View())
	case modeNote:
		body = fmt.Sprintf("🍅 pomo — note\n\n%s\n\n(enter to save · esc cancel)", m.input.View())
	default:
		s := m.status
		tomato := "🍅"
		if m.pulse < 0.5 {
			tomato = "·🍅·"
		}
		mm := int(s.Remaining.Minutes())
		ss := int(s.Remaining.Seconds()) % 60
		state := "running"
		if s.Paused {
			state = "paused"
		} else if s.Remaining <= 0 {
			state = "done — break time"
		}
		body = fmt.Sprintf("%s pomo\n\n%s\n%02d:%02d  %s\n%s\n\n(p pause · n note · e +5m · d done · s stop · q quit)",
			tomato, s.Session.Topic, mm, ss, state, m.bar.View())
	}
	card := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 3).Render(body)
	return tea.NewView(card)
}
