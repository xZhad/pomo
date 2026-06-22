package store

import "testing"

func TestConfigDefaultsAndSave(t *testing.T) {
	s := newStore(t)
	c, err := s.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.WorkMin != 25 || c.BreakMin != 5 || c.Goal != 4 || !c.Notify {
		t.Fatalf("defaults wrong: %+v", c)
	}
	c.WorkMin = 50
	c.Notify = false
	if err := s.SaveConfig(c); err != nil {
		t.Fatal(err)
	}
	got, _ := s.LoadConfig()
	if got.WorkMin != 50 || got.Notify {
		t.Errorf("persisted config wrong: %+v", got)
	}
	// unset fields keep defaults via merge — Goal still 4
	if got.Goal != 4 {
		t.Errorf("Goal = %d, want 4", got.Goal)
	}
}
