package ui

import (
	"fmt"
	"time"

	"github.com/NiladriHazra/filerepo/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the terminal UI.
func Run(initialURL string, cfg config.Config, options RunOptions) error {
	model := newModel(initialURL, cfg, options)
	if stringsTrimmed(initialURL) != "" {
		model.mode = modeLoading
		model.statusMessage = "Parsing URL..."
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run filerepo TUI: %w", err)
	}

	return nil
}

func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd()}
	if stringsTrimmed(m.urlInput) != "" && m.mode == modeLoading {
		cmds = append(cmds, loadRepoCmd(m.urlInput, m.sessionToken, m.configState.Cache.Enabled, m.configState.CacheTTL()))
	}
	return tea.Batch(cmds...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
