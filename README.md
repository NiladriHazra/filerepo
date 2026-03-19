# filerepo

![npm](https://img.shields.io/npm/v/@bytehumi/filerepo?color=blue) ![license](https://img.shields.io/badge/license-MIT-blue) ![Go](https://img.shields.io/badge/go-1.22+-00ADD8)

> Grab any file or folder from GitHub. No clones. Just what you need.

![filerepo](https://pub-51091dcf1e9d4b04bb2e74f489c4f346.r2.dev/76e517e50d0ae0ab208ce38d64e4d96ba4ca5676fd62b2034456747fef585f6d.png)

A terminal UI that lets you browse any GitHub repository and download exactly the files you want — without cloning the entire thing.

## What's New

The current Go version adds quite a bit beyond the original "browse and download files" flow:

- Open branches, tags, commits, compare URLs, and pull requests directly
- Switch branches and tags inside the TUI
- Browse and download GitHub release assets
- View repo metadata, README content, releases, and auth/rate-limit status in-app
- Use named token profiles, import tokens from `gh auth token`, and switch profiles per run
- Keep recent repos and pinned favorites
- Reopen repositories faster with cached repo trees
- Search with fuzzy matching plus filters like `ext:go` and `type:dir`
- Copy GitHub URLs, raw file URLs, and planned local output paths from the TUI
- Choose download conflict handling: `skip`, `overwrite`, `rename`, or `resume`
- Save selections as regular files, `.zip`, or `.tar.gz`
- Write a download manifest beside completed downloads
- Use a better preview with file-type badges, line numbers, and wrap toggle

## Install

```bash
# npm
npm install -g @bytehumi/filerepo

# go
go install github.com/NiladriHazra/filerepo/cmd/filerepo@latest

# or build from source
git clone https://github.com/NiladriHazra/filerepo.git
cd filerepo
go build -o bin/filerepo ./cmd/filerepo
```

## Usage

```bash
# launch the TUI
filerepo

# open a specific repo
filerepo https://github.com/torvalds/linux

# open a specific branch, tag, or commit
filerepo https://github.com/torvalds/linux --ref master
filerepo https://github.com/torvalds/linux/commit/abc123def456

# open a specific folder
filerepo https://github.com/rust-lang/rust/tree/master/src/tools

# browse changed files in a compare or pull request
filerepo https://github.com/owner/repo/compare/main...feature
filerepo https://github.com/owner/repo/pull/42

# download to current directory without a subfolder
filerepo https://github.com/user/repo --cwd --no-folder

# use a GitHub token for private repos
filerepo --token ghp_xxxxxxxxxxxx https://github.com/user/private-repo

# use a saved token profile
filerepo --profile work https://github.com/user/private-repo
```

## How it works

1. You paste a GitHub URL
2. filerepo fetches the entire file tree with **one API call**
3. You browse, search, and select files in the TUI
4. Selected files are downloaded in parallel (up to 8 at a time)

No git needed. No cloning. It talks directly to the GitHub API.

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `Enter` / `l` / `Right` | Enter directory |
| `Backspace` / `h` / `Left` | Go back |
| `j` / `k` / `Up` / `Down` | Navigate |
| `Space` | Toggle selection |
| `a` | Select all |
| `u` | Unselect all |
| `d` | Download selected (with conflict/archive options) |
| `/` | Search with fuzzy matching and filters such as `ext:go` or `type:dir` |
| `b` / `t` | Switch branch or tag |
| `R` | Browse release assets |
| `m` | Open repo summary / README / releases overlay |
| `f` | Toggle favorite repository |
| `y` / `Y` / `P` | Copy GitHub URL / raw URL / planned output path |
| `i` | Toggle ASCII/icon mode |
| `g` / `G` | Jump to top/bottom |
| `Esc` | Exit search / go home |
| `q` / `Ctrl+C` | Quit |

### Preview shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll preview |
| `w` | Toggle line wrapping |
| `n` | Toggle line numbers |
| `Esc` / `q` | Close preview |

### Save prompt shortcuts

| Key | Action |
|-----|--------|
| `s` | Conflict mode: skip |
| `o` | Conflict mode: overwrite |
| `r` | Conflict mode: rename |
| `e` | Conflict mode: resume |
| `f` | Output mode: regular files |
| `z` | Output mode: zip archive |
| `t` | Output mode: tar.gz archive |

### Overlay shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Move between repo summary tabs |
| `R` | Open release asset picker |
| `y` | Copy selected release asset URL |

## Configuration

```bash
# save a GitHub token (for private repos + higher rate limits)
filerepo config set token YOUR_GITHUB_TOKEN

# manage named token profiles
filerepo config profile set-token work YOUR_GITHUB_TOKEN
filerepo config profile use work
filerepo config profile import-gh work
filerepo config profile list
filerepo config profile unset-token work

# set a custom download directory
filerepo config set path /your/preferred/path

# manage favorites and recents
filerepo config favorite add https://github.com/owner/repo
filerepo config favorite remove https://github.com/owner/repo
filerepo config favorite list
filerepo config recent list
filerepo config recent clear

# view current config
filerepo config list

# remove settings
filerepo config unset token
filerepo config unset path
```

## CLI flags

| Flag | Description |
|------|-------------|
| `--cwd` | Download to current working directory |
| `--no-folder` | Don't create a repo-named subfolder |
| `--token <TOKEN>` | One-time GitHub token (not saved) |
| `--profile <NAME>` | Use a saved token profile |
| `--ref <REF>` | Open a branch, tag, or commit SHA |
| `--help`, `-h` | Show help |
| `--version`, `-v` | Print version |

## Caching

- Repo tree caching is enabled by default
- Cache entries are keyed by `owner/repo/ref`
- The default cache TTL is `30m`
- Cache files live under your filerepo config directory

## Download behavior

- Regular file downloads still run in parallel
- If a file already exists, you can choose `skip`, `overwrite`, `rename`, or `resume`
- You can save a selection as regular files, a `.zip`, or a `.tar.gz`
- filerepo writes a JSON manifest beside successful downloads so you can track:
  repository URL, ref, output mode, conflict mode, output path, and selected paths

## Features

- Browse any public GitHub repo without cloning
- Open repository roots, subfolders, branches, tags, commits, compare URLs, and pull requests
- Search across the entire file tree with fuzzy matching and `ext:` / `type:` filters
- Select individual files or entire folders
- Parallel downloads (8 concurrent)
- Progress bar with file-by-file status
- LFS file support
- Branch/tag switcher inside the TUI
- Preview with line numbers, wrap toggle, and file-type badges
- Repo summary overlay with README, releases, and auth/rate-limit status
- Release asset browser and downloader
- Compare view and pull-request file browsing
- Token profiles plus `gh auth token` import
- Recent repos and favorites
- Cached repo trees for faster reopen
- Download conflict strategies: skip, overwrite, rename, resume
- Archive output modes: regular files, `.zip`, `.tar.gz`
- Download manifest written alongside each completed download
- Copy actions for GitHub URLs, raw file URLs, and planned output paths
- Auto-detects local git remote as default URL
- Go-based CLI and TUI implementation
- Dirs-first sorted file listing
- Fallback to folder-by-folder mode for massive repos

## License

MIT

---

Built by [byteHumi](https://github.com/NiladriHazra)
