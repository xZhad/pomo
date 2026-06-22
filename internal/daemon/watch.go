package daemon

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/xZhad/pomo/internal/notify"
	"github.com/xZhad/pomo/internal/store"
)

// Watch polls the statefile until the active session `id` reaches its deadline,
// then fires a notification. Returns early (nil) if the active session changes
// or is cleared. maxIters > 0 bounds the loop (tests); 0 = unbounded.
func Watch(s *store.Store, n notify.Notifier, id string, now func() time.Time, tick time.Duration, maxIters int) error {
	for i := 0; maxIters == 0 || i < maxIters; i++ {
		st, ok, err := s.LoadState()
		if err != nil {
			return err
		}
		if !ok || st.ID != id {
			return nil // stale: session ended or replaced
		}
		if !st.Paused && !now().Before(st.Deadline) {
			return n.Notify("pomo", "Time's up — break time 🍅")
		}
		time.Sleep(tick)
	}
	return nil
}

// Spawn re-execs the current binary as a detached `pomo _watch <id>` process.
func Spawn(id string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "_watch", id)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start() // not waited on — detached
}
