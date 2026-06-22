package model

import (
	"time"

	"github.com/oklog/ulid/v2"
)

type Note struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

type Session struct {
	ID        string     `json:"id"`
	Topic     string     `json:"topic"`
	Duration  int        `json:"duration"` // planned seconds
	Started   time.Time  `json:"started"`
	Ended     *time.Time `json:"ended,omitempty"`
	Completed bool       `json:"completed"`
	Tags      []string   `json:"tags,omitempty"`
	Notes     []Note     `json:"notes,omitempty"`
	XP        int        `json:"xp,omitempty"`
}

// NewID returns a fresh ulid string.
func NewID() string { return ulid.Make().String() }
