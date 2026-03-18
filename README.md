# filerepo

![npm](https://img.shields.io/npm/v/@bytehumi/filerepo?color=blue) ![license](https://img.shields.io/badge/license-MIT-blue) ![Go](https://img.shields.io/badge/go-1.22+-00ADD8)

> Grab any file or folder from GitHub. No clones. Just what you need.

![filerepo](https://pub-51091dcf1e9d4b04bb2e74f489c4f346.r2.dev/5588ae345d0f29a8228676712ed2acb28202b256504f1f29980e77065a3510d7.png)

A terminal UI that lets you browse any GitHub repository and download exactly the files you want — without cloning the entire thing.

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

# open a specific folder
filerepo https://github.com/rust-lang/rust/tree/master/src/tools

# download to current directory without a subfolder
filerepo https://github.com/user/repo --cwd --no-folder

# use a GitHub token for private repos
filerepo --token ghp_xxxxxxxxxxxx https://github.com/user/private-repo
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
| `d` | Download selected |
| `/` | Search across all files |
| `i` | Toggle ASCII/icon mode |
| `g` / `G` | Jump to top/bottom |
| `Esc` | Exit search / go home |
| `q` / `Ctrl+C` | Quit |

## Configuration

```bash
# save a GitHub token (for private repos + higher rate limits)
filerepo config set token YOUR_GITHUB_TOKEN

# set a custom download directory
filerepo config set path /your/preferred/path

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

## Features

- Browse any public GitHub repo without cloning
- Search across the entire file tree
- Select individual files or entire folders
- Parallel downloads (8 concurrent)
- Progress bar with file-by-file status
- LFS file support
- Auto-detects local git remote as default URL
- Go-based CLI and TUI implementation
- Dirs-first sorted file listing
- Fallback to folder-by-folder mode for massive repos

## License

MIT

---

Built by [byteHumi](https://github.com/NiladriHazra)
