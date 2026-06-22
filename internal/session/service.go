package session

import (
	"errors"
	"time"

	"github.com/xZhad/pomo/internal/model"
	"github.com/xZhad/pomo/internal/notify"
	"github.com/xZhad/pomo/internal/store"
)

type Service struct {
	Store    *store.Store
	Now      func() time.Time
	IDGen    func() string
	Notifier notify.Notifier
}

func New(s *store.Store) *Service {
	return &Service{Store: s, Now: time.Now, IDGen: model.NewID, Notifier: notify.Beep{}}
}

var ErrActive = errors.New("a session is already active")
var ErrNoActive = errors.New("no active session")

type StartOpts struct {
	Topic   string
	WorkMin int
	Tags    []string
}

func (svc *Service) Start(opts StartOpts) (model.Session, error) {
	if _, ok, err := svc.Store.LoadState(); err != nil {
		return model.Session{}, err
	} else if ok {
		return model.Session{}, ErrActive
	}
	workMin := opts.WorkMin
	if workMin <= 0 {
		cfg, err := svc.Store.LoadConfig()
		if err != nil {
			return model.Session{}, err
		}
		workMin = cfg.WorkMin
	}
	now := svc.Now().UTC()
	sess := model.Session{
		ID:       svc.IDGen(),
		Topic:    opts.Topic,
		Duration: workMin * 60,
		Started:  now,
		Tags:     opts.Tags,
	}
	if err := svc.Store.AppendSession(sess); err != nil {
		return model.Session{}, err
	}
	st := store.State{ID: sess.ID, Started: now, Deadline: now.Add(time.Duration(sess.Duration) * time.Second)}
	if err := svc.Store.SaveState(st); err != nil {
		return model.Session{}, err
	}
	return sess, nil
}

type Status struct {
	Active    bool
	Session   model.Session
	Remaining time.Duration
	Paused    bool
}

func (svc *Service) Status() (Status, error) {
	st, ok, err := svc.Store.LoadState()
	if err != nil || !ok {
		return Status{Active: false}, err
	}
	ref := svc.Now().UTC()
	if st.Paused {
		ref = st.PausedAt
	}
	rem := st.Deadline.Sub(ref)
	if rem < 0 {
		rem = 0
	}
	out := Status{Active: true, Remaining: rem, Paused: st.Paused}
	all, err := svc.Store.AllSessions()
	if err != nil {
		return out, err
	}
	for _, s := range all {
		if s.ID == st.ID {
			out.Session = s
			break
		}
	}
	return out, nil
}

func (svc *Service) activeID() (string, error) {
	st, ok, err := svc.Store.LoadState()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNoActive
	}
	return st.ID, nil
}

func (svc *Service) Note(text string) error {
	id, err := svc.activeID()
	if err != nil {
		return err
	}
	at := svc.Now().UTC()
	n, err := svc.Store.UpdateSession(id, func(s model.Session) model.Session {
		s.Notes = append(s.Notes, model.Note{At: at, Text: text})
		return s
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoActive
	}
	return nil
}

func (svc *Service) finish(completed bool) (model.Session, error) {
	id, err := svc.activeID()
	if err != nil {
		return model.Session{}, err
	}
	end := svc.Now().UTC()
	if _, err := svc.Store.UpdateSession(id, func(s model.Session) model.Session {
		s.Ended = &end
		s.Completed = completed
		return s
	}); err != nil {
		return model.Session{}, err
	}
	if err := svc.Store.ClearState(); err != nil {
		return model.Session{}, err
	}
	all, err := svc.Store.AllSessions()
	if err != nil {
		return model.Session{}, err
	}
	for _, s := range all {
		if s.ID == id {
			return s, nil
		}
	}
	return model.Session{}, nil
}

func (svc *Service) Done() (model.Session, error) { return svc.finish(true) }
func (svc *Service) Stop() (model.Session, error) { return svc.finish(false) }
