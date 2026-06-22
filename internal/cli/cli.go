package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xZhad/pomo/internal/daemon"
	"github.com/xZhad/pomo/internal/model"
	"github.com/xZhad/pomo/internal/notify"
	"github.com/xZhad/pomo/internal/report"
	"github.com/xZhad/pomo/internal/session"
	"github.com/xZhad/pomo/internal/store"
	"github.com/xZhad/pomo/internal/tui"
)

func Run(args []string, out io.Writer) int {
	if len(args) == 0 || args[0] == "tui" {
		st, err := store.Open()
		if err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		svcTUI := session.New(st)
		if cfg, err := st.LoadConfig(); err == nil && !cfg.Notify {
			svcTUI.Notifier = notify.Noop{}
		}
		if err := tui.Run(svcTUI); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		return 0
	}
	st, err := store.Open()
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	svc := session.New(st)
	cmd, rest := args[0], args[1:]

	switch cmd {
	case "start":
		return cmdStart(svc, rest, out)
	case "status":
		return cmdStatus(svc, out)
	case "note":
		if err := svc.Note(joinArgs(rest)); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		fmt.Fprintln(out, "note saved")
		return 0
	case "done":
		return finish(out, svc.Done, "done")
	case "stop":
		return finish(out, svc.Stop, "stopped")
	case "pause":
		return wrap(out, svc.Pause())
	case "resume":
		return wrap(out, svc.Resume())
	case "extend":
		return cmdExtend(svc, rest, out)
	case "last":
		return cmdLast(st, out)
	case "log", "history":
		return cmdLog(st, rest, out)
	case "topics":
		return cmdReport(st, "topic", out)
	case "report":
		return cmdReportFlag(st, rest, out)
	case "rm":
		return cmdRm(st, rest, out)
	case "config":
		return cmdConfig(st, rest, out)
	case "_watch":
		if len(rest) == 0 {
			return 2
		}
		var wn notify.Notifier = notify.Beep{}
		if cfg, err := st.LoadConfig(); err == nil && !cfg.Notify {
			wn = notify.Noop{}
		}
		_ = daemon.Watch(st, wn, rest[0], time.Now, time.Second, 0)
		return 0
	default:
		fmt.Fprintln(out, usage)
		return 2
	}
}

const usage = "usage: pomo <start|status|note|done|stop|pause|resume|extend|last|log|topics|report|rm|config>"

func wrap(out io.Writer, err error) int {
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintln(out, "ok")
	return 0
}

func joinArgs(a []string) string { return strings.Join(a, " ") }

func cmdStart(svc *session.Service, args []string, out io.Writer) int {
	var topic string
	var tags []string
	work := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--work":
			if i+1 < len(args) {
				work, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-t", "--tag":
			if i+1 < len(args) {
				tags = append(tags, args[i+1])
				i++
			}
		default:
			if topic == "" {
				topic = args[i]
			}
		}
	}
	if topic == "" {
		fmt.Fprintln(out, "error: topic required")
		return 2
	}
	sess, err := svc.Start(session.StartOpts{Topic: topic, WorkMin: work, Tags: tags})
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if os.Getenv("POMO_NO_SPAWN") != "1" {
		_ = daemon.Spawn(sess.ID)
	}
	fmt.Fprintf(out, "started: %s (%dm) 🍅\n", sess.Topic, sess.Duration/60)
	return 0
}

func cmdStatus(svc *session.Service, out io.Writer) int {
	s, err := svc.Status()
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if !s.Active {
		fmt.Fprintln(out, "no active session")
		return 0
	}
	mm := int(s.Remaining.Minutes())
	ss := int(s.Remaining.Seconds()) % 60
	state := ""
	if s.Paused {
		state = " (paused)"
	}
	fmt.Fprintf(out, "%s — %02d:%02d remaining%s\n", s.Session.Topic, mm, ss, state)
	return 0
}

func cmdExtend(svc *session.Service, args []string, out io.Writer) int {
	mins := 5
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil {
			mins = v
		}
	}
	return wrap(out, svc.Extend(time.Duration(mins)*time.Minute))
}

func finish(out io.Writer, fn func() (model.Session, error), label string) int {
	sess, err := fn()
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "%s: %s\n", label, sess.Topic)
	return 0
}

func cmdLast(st *store.Store, out io.Writer) int {
	s, ok, err := report.Last(st)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(out, "no sessions yet")
		return 0
	}
	fmt.Fprintf(out, "%s\n", s.Topic)
	for _, n := range s.Notes {
		fmt.Fprintf(out, "  %s  %s\n", n.At.Format("15:04"), n.Text)
	}
	return 0
}

func cmdLog(st *store.Store, args []string, out io.Writer) int {
	filter := joinArgs(args)
	sessions, err := report.Log(st, filter)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	for _, s := range sessions {
		fmt.Fprintf(out, "%s  %-20s  %dm\n", s.Started.Format("2006-01-02 15:04"), s.Topic, s.Duration/60)
	}
	return 0
}

func cmdReportFlag(st *store.Store, args []string, out io.Writer) int {
	by := "topic"
	for i := 0; i < len(args); i++ {
		if args[i] == "--by" && i+1 < len(args) {
			by = args[i+1]
			i++
		}
	}
	switch by {
	case "topic", "day", "tag":
		// valid
	default:
		fmt.Fprintf(out, "error: unknown --by value %q (want: topic, day, tag)\n", by)
		return 2
	}
	return cmdReport(st, by, out)
}

func cmdReport(st *store.Store, by string, out io.Writer) int {
	buckets, err := report.Report(st, by)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	for _, b := range buckets {
		fmt.Fprintf(out, "%-20s  %d sessions  %dm\n", b.Key, b.Count, b.TotalSeconds/60)
	}
	return 0
}

func cmdRm(st *store.Store, args []string, out io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(out, "error: rm needs an id")
		return 2
	}
	n, err := st.DeleteSession(args[0])
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "deleted %d\n", n)
	return 0
}

func cmdConfig(st *store.Store, args []string, out io.Writer) int {
	cfg, err := st.LoadConfig()
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if len(args) >= 3 && args[0] == "set" {
		key, val := args[1], args[2]
		switch key {
		case "work":
			cfg.WorkMin, _ = strconv.Atoi(val)
		case "break":
			cfg.BreakMin, _ = strconv.Atoi(val)
		case "goal":
			cfg.Goal, _ = strconv.Atoi(val)
		case "notify":
			cfg.Notify = val == "true"
		default:
			fmt.Fprintln(out, "error: unknown config key:", key)
			return 2
		}
		if err := st.SaveConfig(cfg); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	}
	fmt.Fprintf(out, "work=%d break=%d goal=%d notify=%v\n", cfg.WorkMin, cfg.BreakMin, cfg.Goal, cfg.Notify)
	return 0
}
