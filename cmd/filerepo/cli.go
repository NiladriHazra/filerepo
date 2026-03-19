package main

import (
	"fmt"
	"strings"

	"github.com/NiladriHazra/filerepo/internal/config"
	gh "github.com/NiladriHazra/filerepo/internal/github"
	"github.com/NiladriHazra/filerepo/internal/ui"
)

const version = "1.1.0"

type rootOptions struct {
	URL      string
	Ref      string
	Profile  string
	CWD      bool
	NoFolder bool
	Token    string
	ShowHelp bool
	ShowVer  bool
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "config" {
		command, err := parseConfigArgs(args[1:])
		if err != nil {
			return err
		}
		return handleConfig(command)
	}

	options, err := parseRootArgs(args)
	if err != nil {
		return err
	}

	switch {
	case options.ShowHelp:
		printHelp()
		return nil
	case options.ShowVer:
		fmt.Println(version)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if options.Profile != "" {
		if err := cfg.SetActiveProfile(options.Profile); err != nil {
			return err
		}
	}

	token := options.Token
	if token == "" {
		token = cfg.ActiveToken()
	}

	initialURL, err := applyRootOptions(options)
	if err != nil {
		return err
	}

	return ui.Run(initialURL, cfg, ui.RunOptions{
		Token:         token,
		ActiveProfile: cfg.ActiveProfileName(),
		CWD:           options.CWD,
		NoFolder:      options.NoFolder,
	})
}

func parseRootArgs(args []string) (rootOptions, error) {
	var options rootOptions

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--cwd":
			options.CWD = true
		case arg == "--no-folder":
			options.NoFolder = true
		case arg == "--help" || arg == "-h":
			options.ShowHelp = true
		case arg == "--version" || arg == "-v":
			options.ShowVer = true
		case arg == "--token":
			index++
			if index >= len(args) {
				return rootOptions{}, fmt.Errorf("--token requires a value")
			}
			options.Token = args[index]
		case strings.HasPrefix(arg, "--token="):
			options.Token = strings.TrimPrefix(arg, "--token=")
		case arg == "--profile":
			index++
			if index >= len(args) {
				return rootOptions{}, fmt.Errorf("--profile requires a value")
			}
			options.Profile = args[index]
		case strings.HasPrefix(arg, "--profile="):
			options.Profile = strings.TrimPrefix(arg, "--profile=")
		case arg == "--ref":
			index++
			if index >= len(args) {
				return rootOptions{}, fmt.Errorf("--ref requires a value")
			}
			options.Ref = args[index]
		case strings.HasPrefix(arg, "--ref="):
			options.Ref = strings.TrimPrefix(arg, "--ref=")
		case strings.HasPrefix(arg, "-"):
			return rootOptions{}, fmt.Errorf("unknown flag: %s", arg)
		case options.URL == "":
			options.URL = arg
		default:
			return rootOptions{}, fmt.Errorf("unexpected argument: %s", arg)
		}
	}

	return options, nil
}

func applyRootOptions(options rootOptions) (string, error) {
	if strings.TrimSpace(options.URL) == "" || strings.TrimSpace(options.Ref) == "" {
		return options.URL, nil
	}

	target, err := gh.ParseURL(options.URL)
	if err != nil {
		return "", err
	}
	target.Branch = options.Ref
	return target.WebURL(), nil
}

func printHelp() {
	fmt.Println(strings.TrimSpace(`
filerepo v1.1.0

Usage:
  filerepo [URL] [--cwd] [--no-folder] [--token TOKEN] [--profile NAME] [--ref REF]
  filerepo config ...

Examples:
  filerepo
  filerepo https://github.com/torvalds/linux
  filerepo https://github.com/rust-lang/rust/tree/master/src/tools --ref master
  filerepo --profile work https://github.com/user/private-repo

Flags:
  --cwd            download into the current working directory
  --no-folder      do not create a repo-named subdirectory
  --token TOKEN    use a one-off GitHub token for this run
  --profile NAME   use a saved token profile for this run
  --ref REF        open a branch, tag, or commit SHA directly
  --help, -h       show help
  --version, -v    print version

Config:
  filerepo config list
  filerepo config set token TOKEN
  filerepo config set path /downloads
  filerepo config profile list
  filerepo config profile use work
  filerepo config profile set-token work TOKEN
  filerepo config profile import-gh work
  filerepo config favorite add https://github.com/owner/repo
  filerepo config recent clear
`))
}
