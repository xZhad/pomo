package notify

import "testing"

func TestRecorder(t *testing.T) {
	var n Notifier = &Recorder{}
	if err := n.Notify("Break time", "done"); err != nil {
		t.Fatal(err)
	}
	r := n.(*Recorder)
	if len(r.Calls) != 1 || r.Calls[0] != "Break time|done" {
		t.Errorf("calls = %v", r.Calls)
	}
}

func TestNoop(t *testing.T) {
	var n Notifier = Noop{}
	if err := n.Notify("x", "y"); err != nil {
		t.Errorf("noop returned err: %v", err)
	}
}
