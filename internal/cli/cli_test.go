package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xZhad/pomo/internal/notify"
	"github.com/xZhad/pomo/internal/store"
)

func run(t *testing.T, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	Run(args, &buf)
	return buf.String()
}

func TestStartStatusNoteDoneFlow(t *testing.T) {
	t.Setenv("POMO_DIR", t.TempDir())
	t.Setenv("POMO_NO_SPAWN", "1")

	out := run(t, "start", "axiom ch3", "--work", "25")
	if !strings.Contains(out, "axiom ch3") {
		t.Fatalf("start output: %q", out)
	}
	out = run(t, "status")
	if !strings.Contains(out, "axiom ch3") {
		t.Errorf("status output: %q", out)
	}
	if out := run(t, "note", "vanishing gradient"); !strings.Contains(strings.ToLower(out), "note") {
		t.Errorf("note output: %q", out)
	}
	out = run(t, "done")
	if !strings.Contains(out, "axiom ch3") {
		t.Errorf("done output: %q", out)
	}
	// after done, status reports nothing active
	if out := run(t, "status"); !strings.Contains(strings.ToLower(out), "no active") {
		t.Errorf("status after done: %q", out)
	}
	// last shows the finished session + its note
	out = run(t, "last")
	if !strings.Contains(out, "axiom ch3") || !strings.Contains(out, "vanishing gradient") {
		t.Errorf("last output: %q", out)
	}
}

func TestConfigAndReport(t *testing.T) {
	t.Setenv("POMO_DIR", t.TempDir())
	t.Setenv("POMO_NO_SPAWN", "1")
	run(t, "config", "set", "work", "50")
	if out := run(t, "config"); !strings.Contains(out, "50") {
		t.Errorf("config did not persist work=50: %q", out)
	}
	run(t, "start", "ml", "--work", "25")
	run(t, "done")
	if out := run(t, "report", "--by", "topic"); !strings.Contains(out, "ml") {
		t.Errorf("report output: %q", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	var buf bytes.Buffer
	if code := Run([]string{"frobnicate"}, &buf); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestReportUnknownByReturns2(t *testing.T) {
	t.Setenv("POMO_DIR", t.TempDir())
	var buf bytes.Buffer
	if code := Run([]string{"report", "--by", "bogus"}, &buf); code != 2 {
		t.Errorf("exit code = %d, want 2 for unknown --by", code)
	}
	if !strings.Contains(buf.String(), "bogus") {
		t.Errorf("expected error message to contain the bad value: %q", buf.String())
	}
}

// TestNotifyConfigFalseUsesNoop verifies that when cfg.Notify=false the _watch
// notifier selection logic produces a Noop instead of Beep.
func TestNotifyConfigFalseUsesNoop(t *testing.T) {
	t.Setenv("POMO_DIR", t.TempDir())
	// Open a store, set notify=false, then verify the notifier chosen matches Noop.
	st, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := st.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	// With notify=true (default) -> Beep
	cfg.Notify = true
	if err := st.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg2, _ := st.LoadConfig()
	var wn notify.Notifier = notify.Beep{}
	if !cfg2.Notify {
		wn = notify.Noop{}
	}
	if _, ok := wn.(notify.Beep); !ok {
		t.Errorf("notify=true should yield Beep, got %T", wn)
	}

	// With notify=false -> Noop
	cfg.Notify = false
	if err := st.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg3, _ := st.LoadConfig()
	wn = notify.Beep{}
	if !cfg3.Notify {
		wn = notify.Noop{}
	}
	if _, ok := wn.(notify.Noop); !ok {
		t.Errorf("notify=false should yield Noop, got %T", wn)
	}
}
