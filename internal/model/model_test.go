package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSessionJSONRoundTrip(t *testing.T) {
	s := Session{
		ID: "01J", Topic: "ml", Duration: 1500,
		Started:   time.Date(2026, 6, 8, 18, 30, 0, 0, time.UTC),
		Completed: true,
		Tags:      []string{"study"},
		Notes:     []Note{{At: time.Date(2026, 6, 8, 18, 38, 0, 0, time.UTC), Text: "hi"}},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got Session
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Topic != "ml" || got.Duration != 1500 || !got.Completed || got.Notes[0].Text != "hi" {
		t.Errorf("round trip mismatch: %+v", got)
	}
	// omitempty: an empty session must not emit ended/tags/notes/xp
	min, _ := json.Marshal(Session{ID: "x"})
	if s := string(min); contains(s, "ended") || contains(s, "tags") || contains(s, "xp") {
		t.Errorf("omitempty failed: %s", s)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestNewIDUnique(t *testing.T) {
	if NewID() == "" {
		t.Error("empty id")
	}
	if NewID() == NewID() {
		t.Error("ids not unique")
	}
}
