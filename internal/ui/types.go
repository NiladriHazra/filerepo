package ui

import (
	"time"

	"github.com/NiladriHazra/filerepo/internal/download"
	gh "github.com/NiladriHazra/filerepo/internal/github"
)

type mode int

const (
	modeInput mode = iota
	modeLoading
	modeBrowse
)

type previewStatus int

const (
	previewLoading previewStatus = iota
	previewReady
	previewError
)

type toastKind int

const (
	toastInfo toastKind = iota
	toastSuccess
	toastWarning
	toastError
)

type toast struct {
	message   string
	kind      toastKind
	expiresAt time.Time
}

type previewState struct {
	path    string
	content string
	scroll  int
	status  previewStatus
}

type savePrompt struct {
	input     string
	cursor    int
	itemCount int
}

type navState struct {
	url    gh.URL
	cursor int
}

type model struct {
	width  int
	height int
	frame  int

	mode          mode
	urlInput      string
	urlCursor     int
	statusMessage string

	client                 *gh.Client
	sessionToken           string
	configuredDownloadPath string
	cwd                    bool
	noFolder               bool

	currentURL *gh.URL
	items      []gh.RepoItem

	cursor       int
	scrollOffset int
	navigation   []navState

	fullTree    []gh.RepoItem
	hasFullTree bool
	folderSizes map[string]uint64

	searching    bool
	searchQuery  string
	selectedPath map[string]struct{}
	asciiMode    bool

	preview            *previewState
	savePrompt         *savePrompt
	downloadPathChoice string
	downloading        bool
	downloadProgress   *download.Progress

	toast *toast
}
