package report

import (
	"testing"
	"time"

	"github.com/xZhad/pomo/internal/model"
	"github.com/xZhad/pomo/internal/store"
)

func seed(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("POMO_DIR", t.TempDir())
	s, _ := store.Open()
	day1 := time.Date(2026, 6, 8, 18, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 9, 9, 0, 0, 0, time.UTC)
	s.AppendSession(model.Session{ID: "a", Topic: "ml", Duration: 1500, Started: day1, Tags: []string{"study", "ai"}, Completed: true})
	s.AppendSession(model.Session{ID: "b", Topic: "go", Duration: 900, Started: day1.Add(time.Hour), Tags: []string{"study"}, Completed: true})
	s.AppendSession(model.Session{ID: "c", Topic: "ml", Duration: 1200, Started: day2, Tags: []string{"ai"}, Completed: true})
	return s
}

func TestLastAndLog(t *testing.T) {
	s := seed(t)
	last, ok, err := Last(s)
	if err != nil || !ok || last.ID != "c" {
		t.Fatalf("Last = %+v ok=%v err=%v", last, ok, err)
	}
	all, err := Log(s, "")
	if err != nil || len(all) != 3 {
		t.Fatalf("Log all len=%d err=%v", len(all), err)
	}
	if all[0].ID != "c" {
		t.Errorf("Log not sorted desc: first=%s", all[0].ID)
	}
	mlOnly, err := Log(s, "topic=ml")
	if err != nil || len(mlOnly) != 2 {
		t.Errorf("Log filtered len=%d err=%v", len(mlOnly), err)
	}
}

func TestReports(t *testing.T) {
	s := seed(t)
	topics, err := Topics(s)
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Bucket{}
	for _, b := range topics {
		byKey[b.Key] = b
	}
	if byKey["ml"].TotalSeconds != 2700 || byKey["ml"].Count != 2 {
		t.Errorf("ml bucket = %+v", byKey["ml"])
	}
	if topics[0].Key != "ml" { // sorted by total desc (2700 > 900)
		t.Errorf("topics not sorted: %+v", topics)
	}

	days, err := Report(s, "day")
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 2 {
		t.Errorf("day buckets = %d, want 2", len(days))
	}

	tags, err := Report(s, "tag")
	if err != nil {
		t.Fatal(err)
	}
	tagKey := map[string]Bucket{}
	for _, b := range tags {
		tagKey[b.Key] = b
	}
	if tagKey["study"].Count != 2 || tagKey["ai"].Count != 2 {
		t.Errorf("tag buckets = %+v", tags)
	}
}
