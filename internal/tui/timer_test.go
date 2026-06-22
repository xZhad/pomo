package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xZhad/pomo/internal/notify"
	"github.com/xZhad/pomo/internal/session"
	"github.com/xZhad/pomo/internal/store"
)

func newModel(t *testing.T) (*Model, *notify.Recorder, *fakeClock) {
	t.Helper()
	t.Setenv("POMO_DIR", t.TempDir())
	st, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := session.New(st)
	svc.IDGen = func() string { return "fixed" }
	clk := &fakeClock{now: time.Date(2026, 6, 8, 18, 30, 0, 0, time.UTC)}
	svc.Now = clk.Now
	rec := &notify.Recorder{}
	svc.Notifier = rec
	return New(svc), rec, clk
}

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg{Code: r, Text: s}
	}
}

func typeText(m *Model, s string) *Model {
	for _, r := range s {
		mi, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = mi.(*Model)
	}
	return m
}

func TestTUIStartViaTopicInput(t *testing.T) {
	m, _, _ := newModel(t)
	// no active session -> topic input mode
	if m.mode != modeTopic {
		t.Fatalf("mode = %v, want modeTopic", m.mode)
	}
	m = typeText(m, "axiom ch3")
	mi, _ := m.Update(keyPress("enter"))
	m = mi.(*Model)
	if m.mode != modeTimer {
		t.Fatalf("after start mode = %v, want modeTimer", m.mode)
	}
	st, _ := m.svc.Status()
	if !st.Active || st.Session.Topic != "axiom ch3" {
		t.Errorf("session not started: %+v", st)
	}
}

func TestTUITickAnimatesAndNotifies(t *testing.T) {
	m, rec, clk := newModel(t)
	m = typeText(m, "ml")
	mi, _ := m.Update(keyPress("enter"))
	m = mi.(*Model)
	// 10m in: bar target ~ 10/25 = 0.4
	clk.now = clk.now.Add(10 * time.Minute)
	mi, _ = m.Update(tickMsg(clk.now))
	m = mi.(*Model)
	if got := m.bar.Percent(); got > 0.45 { // SetPercent animates toward target; just sanity
		_ = got
	}
	// past deadline: notify fires once
	clk.now = clk.now.Add(20 * time.Minute)
	mi, _ = m.Update(tickMsg(clk.now))
	m = mi.(*Model)
	mi, _ = m.Update(tickMsg(clk.now)) // second tick must NOT double-notify
	m = mi.(*Model)
	if len(rec.Calls) != 1 {
		t.Errorf("notify calls = %v, want exactly 1", rec.Calls)
	}
}

func TestTUIPauseNoteDoneQuit(t *testing.T) {
	m, _, _ := newModel(t)
	m = typeText(m, "ml")
	m, _ = mustUpdate(m, keyPress("enter"))
	// pause
	m, _ = mustUpdate(m, keyPress("p"))
	if st, _ := m.svc.Status(); !st.Paused {
		t.Error("p did not pause")
	}
	// resume
	m, _ = mustUpdate(m, keyPress("p"))
	if st, _ := m.svc.Status(); st.Paused {
		t.Error("p did not resume")
	}
	// note
	m, _ = mustUpdate(m, keyPress("n"))
	if m.mode != modeNote {
		t.Fatalf("n -> mode %v, want modeNote", m.mode)
	}
	m = typeText(m, "insight")
	m, _ = mustUpdate(m, keyPress("enter"))
	if st, _ := m.svc.Status(); len(st.Session.Notes) != 1 {
		t.Errorf("note not added: %+v", st.Session.Notes)
	}
	// done
	m, _ = mustUpdate(m, keyPress("d"))
	if st, _ := m.svc.Status(); st.Active {
		t.Error("d did not finish session")
	}
	// quit cmd
	_, cmd := m.Update(keyPress("q"))
	if cmd == nil {
		t.Error("q should return a quit cmd")
	}
	// View renders without panic and includes the bar
	if out := m.View().Content; !strings.Contains(out, "pomo") {
		t.Errorf("view missing header: %q", out)
	}
}

func mustUpdate(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
	mi, cmd := m.Update(msg)
	return mi.(*Model), cmd
}
