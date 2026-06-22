package cli

import (
	"bytes"
	"strings"
	"testing"
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
