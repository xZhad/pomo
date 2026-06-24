package session

import (
	"testing"
	"time"

	"github.com/xZhad/pomo/internal/store"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	t.Setenv("POMO_DIR", t.TempDir())
	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := New(s)
	svc.IDGen = func() string { return "fixed-id" }
	return svc
}

func TestStartAndStatus(t *testing.T) {
	svc := newSvc(t)
	base := time.Date(2026, 6, 8, 18, 30, 0, 0, time.UTC)
	svc.Now = func() time.Time { return base }

	sess, err := svc.Start(StartOpts{Topic: "ml", WorkMin: 25, Tags: []string{"study"}})
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID != "fixed-id" || sess.Duration != 1500 || sess.Topic != "ml" {
		t.Fatalf("session wrong: %+v", sess)
	}
	// second start errors (already active)
	if _, err := svc.Start(StartOpts{Topic: "x"}); err == nil {
		t.Error("expected error starting while active")
	}
	// status 10 min later -> 15 min remaining
	svc.Now = func() time.Time { return base.Add(10 * time.Minute) }
	st, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !st.Active || st.Remaining != 15*time.Minute {
		t.Errorf("status = %+v", st)
	}
	if st.Session.Topic != "ml" {
		t.Errorf("status session topic = %q", st.Session.Topic)
	}
	// past deadline -> floored at 0
	svc.Now = func() time.Time { return base.Add(30 * time.Minute) }
	st, _ = svc.Status()
	if st.Remaining != 0 {
		t.Errorf("remaining floored = %v, want 0", st.Remaining)
	}
}

func TestNoteDoneStop(t *testing.T) {
	svc := newSvc(t)
	base := time.Date(2026, 6, 8, 18, 30, 0, 0, time.UTC)
	svc.Now = func() time.Time { return base }
	if _, err := svc.Start(StartOpts{Topic: "ml", WorkMin: 25}); err != nil {
		t.Fatal(err)
	}

	svc.Now = func() time.Time { return base.Add(8 * time.Minute) }
	if err := svc.Note("got vanishing gradient"); err != nil {
		t.Fatal(err)
	}
	svc.Now = func() time.Time { return base.Add(25 * time.Minute) }
	done, err := svc.Done()
	if err != nil {
		t.Fatal(err)
	}
	if !done.Completed || done.Ended == nil || len(done.Notes) != 1 || done.Notes[0].Text != "got vanishing gradient" {
		t.Fatalf("done wrong: %+v", done)
	}
	// state cleared
	if st, _ := svc.Status(); st.Active {
		t.Error("state not cleared after Done")
	}
	// note with no active session errors
	if err := svc.Note("x"); err == nil {
		t.Error("expected ErrNoActive")
	}

	// stop path: start then stop -> not completed
	svc.Now = func() time.Time { return base.Add(time.Hour) }
	if _, err := svc.Start(StartOpts{Topic: "go", WorkMin: 25}); err != nil {
		t.Fatal(err)
	}
	stopped, err := svc.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Completed {
		t.Error("stopped session should not be completed")
	}
	if stopped.Ended == nil {
		t.Error("stopped session should have Ended set")
	}
	if st, _ := svc.Status(); st.Active {
		t.Error("state not cleared after Stop")
	}
}

func TestPauseResumeExtend(t *testing.T) {
	svc := newSvc(t)
	base := time.Date(2026, 6, 8, 18, 30, 0, 0, time.UTC)
	svc.Now = func() time.Time { return base }
	svc.Start(StartOpts{Topic: "ml", WorkMin: 25}) // deadline base+25m

	// pause at +10m -> remaining frozen at 15m
	svc.Now = func() time.Time { return base.Add(10 * time.Minute) }
	if err := svc.Pause(); err != nil {
		t.Fatal(err)
	}
	svc.Now = func() time.Time { return base.Add(20 * time.Minute) } // 10m paused
	if st, _ := svc.Status(); st.Remaining != 15*time.Minute || !st.Paused {
		t.Errorf("paused status = %+v, want 15m & paused", st)
	}
	// resume at +20m -> deadline shifts by 10m to base+35m
	if err := svc.Resume(); err != nil {
		t.Fatal(err)
	}
	if st, _ := svc.Status(); st.Remaining != 15*time.Minute || st.Paused {
		t.Errorf("resumed status = %+v, want 15m & not paused", st)
	}
	// extend +5m -> remaining 20m, session duration grows
	if err := svc.Extend(5 * time.Minute); err != nil {
		t.Fatal(err)
	}
	if st, _ := svc.Status(); st.Remaining != 20*time.Minute {
		t.Errorf("extended remaining = %v, want 20m", st.Remaining)
	}
	if st, _ := svc.Status(); st.Session.Duration != 1500+300 {
		t.Errorf("duration after extend = %d, want 1800", st.Session.Duration)
	}
}

func TestBreakPhaseAndCycle(t *testing.T) {
	svc := newSvc(t)
	// start + complete a focus
	if _, err := svc.Start(StartOpts{Topic: "x", WorkMin: 25}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Done(); err != nil {
		t.Fatal(err)
	}
	n, _ := svc.CompletedFocusToday()
	if n != 1 {
		t.Errorf("completed today = %d, want 1", n)
	}
	// start a break — ephemeral, no session logged, phase set
	if _, err := svc.StartBreak(false); err != nil {
		t.Fatal(err)
	}
	stt, _ := svc.Status()
	if !stt.Active || stt.Phase != "short" {
		t.Errorf("break status = %+v, want active short", stt)
	}
	// break creates no session
	all, _ := svc.Store.AllSessions()
	if len(all) != 1 {
		t.Errorf("sessions = %d, want 1 (break not logged)", len(all))
	}
	// end break clears state
	if err := svc.EndBreak(); err != nil {
		t.Fatal(err)
	}
	if s, _ := svc.Status(); s.Active {
		t.Error("break should be cleared")
	}
}
