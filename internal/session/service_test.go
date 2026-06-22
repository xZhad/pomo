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
