package download

import "time"

// ConflictStrategy controls how filerepo handles an existing destination file.
type ConflictStrategy string

const (
	ConflictSkip      ConflictStrategy = "skip"
	ConflictOverwrite ConflictStrategy = "overwrite"
	ConflictRename    ConflictStrategy = "rename"
	ConflictResume    ConflictStrategy = "resume"
)

// OutputMode controls whether downloads are written as files or archives.
type OutputMode string

const (
	OutputFiles OutputMode = "files"
	OutputZip   OutputMode = "zip"
	OutputTarGz OutputMode = "tar.gz"
)

// Options configures downloader behavior.
type Options struct {
	ConflictStrategy ConflictStrategy
	OutputMode       OutputMode
}

// Result describes the final download artifact.
type Result struct {
	OutputPath string
}

// Manifest records what filerepo downloaded.
type Manifest struct {
	GeneratedAt      time.Time        `json:"generated_at"`
	RepositoryURL    string           `json:"repository_url"`
	Ref              string           `json:"ref"`
	OutputMode       OutputMode       `json:"output_mode"`
	ConflictStrategy ConflictStrategy `json:"conflict_strategy"`
	OutputPath       string           `json:"output_path"`
	SelectedPaths    []string         `json:"selected_paths"`
}
