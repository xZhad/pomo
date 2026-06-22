package notify

import "github.com/gen2brain/beeep"

type Notifier interface {
	Notify(title, body string) error
}

type Beep struct{}

func (Beep) Notify(title, body string) error { return beeep.Notify(title, body, "") }

type Noop struct{}

func (Noop) Notify(title, body string) error { return nil }

type Recorder struct{ Calls []string }

func (r *Recorder) Notify(title, body string) error {
	r.Calls = append(r.Calls, title+"|"+body)
	return nil
}
