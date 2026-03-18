package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the terminal UI.
func Run(initialURL, token, downloadPath string, cwd, noFolder bool) error {
	model := newModel(initialURL, token, downloadPath, cwd, noFolder)
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
		cmds = append(cmds, loadRepoCmd(m.urlInput, m.sessionToken))
	}
	return tea.Batch(cmds...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
