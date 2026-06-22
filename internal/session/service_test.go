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
