package ui

import (
	"strings"

	"github.com/NiladriHazra/filerepo/internal/download"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+w", "ctrl+u", "delete":
		m.urlInput = ""
		m.urlCursor = 0
	case "backspace":
		if m.urlCursor > 0 {
			m.urlInput = m.urlInput[:m.urlCursor-1] + m.urlInput[m.urlCursor:]
			m.urlCursor--
		}
	case "left":
		if m.urlCursor > 0 {
			m.urlCursor--
		}
	case "right":
		if m.urlCursor < len(m.urlInput) {
			m.urlCursor++
		}
	case "home":
		m.urlCursor = 0
	case "end":
		m.urlCursor = len(m.urlInput)
	case "tab":
		target := "https://github.com/"
		if m.urlInput == "" || strings.HasPrefix(target, m.urlInput) {
			m.urlInput = target
			m.urlCursor = len(m.urlInput)
		}
	case "enter":
		if stringsTrimmed(m.urlInput) == "" {
			m.showToast("Please enter a GitHub URL.", toastWarning)
			return m, nil
		}
		m.mode = modeLoading
		m.statusMessage = "Parsing URL..."
		return m, loadRepoCmd(m.urlInput, m.sessionToken, m.configState.Cache.Enabled, m.configState.CacheTTL())
	case "esc":
		return m, tea.Quit
	default:
		if msg.Type == tea.KeyRunes {
			insert := string(msg.Runes)
			left := m.urlInput[:m.urlCursor]
			right := m.urlInput[m.urlCursor:]
			m.urlInput = left + insert + right
			m.urlCursor += len(insert)
		}
	}

	return m, nil
}

func (m *model) handleLoadingMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeInput
		m.statusMessage = ""
	}
	return m, nil
}

func (m *model) handleSavePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.savePrompt = nil
		return m, nil
	case "enter":
		chosenPath := stringsTrimmed(m.savePrompt.input)
		if chosenPath == "" {
			dir, err := defaultDownloadDir()
			if err != nil {
				m.showToast(err.Error(), toastError)
				return m, nil
			}
			chosenPath = dir
		}

		items := m.savePrompt.items
		if len(items) == 0 {
			items = m.selectedOrFocusedItems()
		}
		progress := &download.Progress{Total: len(items)}
		m.downloadProgress = progress
		m.downloading = true
		m.savePrompt = nil

		request := downloadRequest{
			currentURL:   *m.currentURL,
			repoURL:      m.currentRepoURL,
			selected:     items,
			fullTree:     m.fullTree,
			hasFullTree:  m.hasFullTree,
			token:        m.sessionToken,
			configPath:   m.configuredDownloadPath,
			overridePath: chosenPath,
			cwd:          m.cwd,
			noFolder:     m.noFolder,
			conflict:     m.savePrompt.conflict,
			outputMode:   m.savePrompt.output,
			progress:     progress,
		}
		return m, performDownloadCmd(request)
	case "s":
		m.savePrompt.conflict = download.ConflictSkip
	case "o":
		m.savePrompt.conflict = download.ConflictOverwrite
	case "r":
		m.savePrompt.conflict = download.ConflictRename
	case "e":
		m.savePrompt.conflict = download.ConflictResume
	case "f":
		m.savePrompt.output = download.OutputFiles
	case "z":
		m.savePrompt.output = download.OutputZip
	case "t":
		m.savePrompt.output = download.OutputTarGz
	case "ctrl+w", "ctrl+u":
		m.savePrompt.input = ""
		m.savePrompt.cursor = 0
	case "backspace":
		if m.savePrompt.cursor > 0 {
			m.savePrompt.input = m.savePrompt.input[:m.savePrompt.cursor-1] + m.savePrompt.input[m.savePrompt.cursor:]
			m.savePrompt.cursor--
		}
	case "delete":
		if m.savePrompt.cursor < len(m.savePrompt.input) {
			m.savePrompt.input = m.savePrompt.input[:m.savePrompt.cursor] + m.savePrompt.input[m.savePrompt.cursor+1:]
		}
	case "left":
		if m.savePrompt.cursor > 0 {
			m.savePrompt.cursor--
		}
	case "right":
		if m.savePrompt.cursor < len(m.savePrompt.input) {
			m.savePrompt.cursor++
		}
	case "home":
		m.savePrompt.cursor = 0
	case "end":
		m.savePrompt.cursor = len(m.savePrompt.input)
	default:
		if msg.Type == tea.KeyRunes {
			left := m.savePrompt.input[:m.savePrompt.cursor]
			right := m.savePrompt.input[m.savePrompt.cursor:]
			m.savePrompt.input = left + string(msg.Runes) + right
			m.savePrompt.cursor += len(msg.Runes)
		}
	}

	return m, nil
}
