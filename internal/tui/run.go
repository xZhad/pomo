package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/xZhad/pomo/internal/session"
)

// Run launches the interactive TUI for the given service.
func Run(svc *session.Service) error {
	_, err := tea.NewProgram(New(svc)).Run()
	return err
}
