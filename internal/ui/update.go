package ui

import (
	"fmt"
	"time"

	gh "github.com/NiladriHazra/filerepo/internal/github"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tickMsg:
		m.frame++
		if m.toast != nil && time.Now().After(m.toast.expiresAt) {
			m.toast = nil
		}
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.adjustScroll()
		return m, nil
	case repoLoadedMsg:
		m.client = gh.NewClient(typed.sessionToken)
		m.sessionToken = typed.sessionToken
		m.currentURL = &typed.currentURL
		m.currentRepoURL = typed.repoURL
		m.items = typed.items
		m.fullTree = typed.fullTree
		m.hasFullTree = typed.hasFullTree
		m.folderSizes = typed.folderSizes
		m.repoMetadata = typed.metadata
		m.repoRefs = typed.refs
		m.repoReadme = typed.readme
		m.repoReleases = typed.releases
		m.rateLimit = typed.rateLimit
		m.mode = modeBrowse
		m.statusMessage = ""
		m.resetBrowserPosition(typed.cursor)
		m.recordRecentRepo(typed.repoURL)
		switch typed.warning {
		case "":
			m.showToast("Repository loaded.", toastSuccess)
		default:
			m.showToast(typed.warning, toastWarning)
		}
		return m, nil
	case repoLoadFailedMsg:
		m.mode = modeInput
		m.statusMessage = ""
		m.showToast(typed.err.Error(), toastError)
		return m, nil
	case previewLoadedMsg:
		if m.preview != nil && m.preview.path == typed.path {
			m.preview.status = previewReady
			m.preview.content = typed.content
			m.preview.scroll = 0
		}
		return m, nil
	case previewFailedMsg:
		if m.preview != nil && m.preview.path == typed.path {
			m.preview.status = previewError
			m.preview.content = typed.err.Error()
			m.preview.scroll = 0
		}
		return m, nil
	case downloadFinishedMsg:
		m.downloading = false
		m.downloadProgress = nil
		m.downloadPathChoice = ""
		switch {
		case typed.err != nil:
			m.showToast(fmt.Sprintf("Download failed: %v", typed.err), toastError)
		case typed.empty:
			m.showToast("No downloadable files found in the selection.", toastWarning)
		case len(typed.errors) == 0:
			if typed.manifest != "" {
				m.showToast("Downloaded to: "+typed.outputPath+"  [manifest saved]", toastSuccess)
			} else {
				m.showToast("Downloaded to: "+typed.outputPath, toastSuccess)
			}
		default:
			m.showToast(fmt.Sprintf("Download finished with %d errors", len(typed.errors)), toastWarning)
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(typed)
	default:
		return m, nil
	}
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeInput:
		return m.handleInputMode(msg)
	case modeLoading:
		return m.handleLoadingMode(msg)
	case modeBrowse:
		return m.handleBrowseMode(msg)
	default:
		return m, nil
	}
}
