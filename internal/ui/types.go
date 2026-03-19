package ui

import (
	"time"

	"github.com/NiladriHazra/filerepo/internal/config"
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
	path        string
	content     string
	scroll      int
	status      previewStatus
	wrap        bool
	showNumbers bool
}

type savePrompt struct {
	input     string
	cursor    int
	itemCount int
	conflict  download.ConflictStrategy
	output    download.OutputMode
	items     []gh.RepoItem
}

type navState struct {
	url    gh.URL
	cursor int
}

type refPickerState struct {
	kind   string
	query  string
	cursor int
}

type releasePickerState struct {
	cursor int
}

type infoOverlayState struct {
	tab    int
	scroll int
}

// RunOptions controls UI startup behavior.
type RunOptions struct {
	Token         string
	ActiveProfile string
	CWD           bool
	NoFolder      bool
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
	activeProfile          string
	configState            config.Config
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
	refPicker          *refPickerState
	releasePicker      *releasePickerState
	infoOverlay        *infoOverlayState
	savePrompt         *savePrompt
	downloadPathChoice string
	downloading        bool
	downloadProgress   *download.Progress

	repoMetadata   gh.RepoMetadata
	repoRefs       []gh.RepoRef
	repoReadme     gh.Readme
	repoReleases   []gh.Release
	rateLimit      gh.RateLimitStatus
	currentRepoURL string

	toast *toast
}
