package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/NiladriHazra/filerepo/internal/config"
	"github.com/NiladriHazra/filerepo/internal/ui"
)

type rootOptions struct {
	URL      string
	CWD      bool
	NoFolder bool
	Token    string
}

type configCommand struct {
	action string
	target string
	value  string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "filerepo: %v\n", err)
		os.Exit(1)
	}
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

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	token := options.Token
	if token == "" {
		token = cfg.GitHubToken
	}

	return ui.Run(options.URL, token, cfg.DownloadPath, options.CWD, options.NoFolder)
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
		case arg == "--token":
			index++
			if index >= len(args) {
				return rootOptions{}, fmt.Errorf("--token requires a value")
			}
			options.Token = args[index]
		case strings.HasPrefix(arg, "--token="):
			options.Token = strings.TrimPrefix(arg, "--token=")
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

func parseConfigArgs(args []string) (configCommand, error) {
	if len(args) == 0 {
		return configCommand{}, fmt.Errorf("usage: filerepo config [set|unset|list]")
	}

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return configCommand{}, fmt.Errorf("usage: filerepo config list")
		}
		return configCommand{action: "list"}, nil
	case "set":
		if len(args) != 3 {
			return configCommand{}, fmt.Errorf("usage: filerepo config set [token|path] VALUE")
		}
		switch args[1] {
		case "token", "path":
			return configCommand{action: "set", target: args[1], value: args[2]}, nil
		default:
			return configCommand{}, fmt.Errorf("unknown config target: %s", args[1])
		}
	case "unset":
		if len(args) != 2 {
			return configCommand{}, fmt.Errorf("usage: filerepo config unset [token|path]")
		}
		switch args[1] {
		case "token", "path":
			return configCommand{action: "unset", target: args[1]}, nil
		default:
			return configCommand{}, fmt.Errorf("unknown config target: %s", args[1])
		}
	default:
		return configCommand{}, fmt.Errorf("unknown config action: %s", args[0])
	}
}

func handleConfig(command configCommand) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch command.action {
	case "list":
		printConfig(cfg)
		return nil
	case "set":
		return applyConfigSet(cfg, command)
	case "unset":
		return applyConfigUnset(cfg, command)
	default:
		return fmt.Errorf("unsupported config action: %s", command.action)
	}
}

func applyConfigSet(cfg config.Config, command configCommand) error {
	switch command.target {
	case "token":
		cfg.GitHubToken = command.value
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("[+] GitHub token saved.")
		return nil
	case "path":
		if err := config.ValidatePath(command.value); err != nil {
			return err
		}
		cfg.DownloadPath = command.value
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("[+] Download path saved.")
		return nil
	default:
		return fmt.Errorf("unsupported config target: %s", command.target)
	}
}

func applyConfigUnset(cfg config.Config, command configCommand) error {
	switch command.target {
	case "token":
		cfg.GitHubToken = ""
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("[+] GitHub token removed.")
		return nil
	case "path":
		cfg.DownloadPath = ""
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("[+] Download path removed.")
		return nil
	default:
		return fmt.Errorf("unsupported config target: %s", command.target)
	}
}

func printConfig(cfg config.Config) {
	fmt.Println("--- filerepo config ---")
	switch cfg.GitHubToken {
	case "":
		fmt.Println("  Token:         (not set)")
	default:
		fmt.Printf("  Token:         %s\n", maskToken(cfg.GitHubToken))
	}

	switch cfg.DownloadPath {
	case "":
		fmt.Println("  Download Path: (default current working directory)")
	default:
		fmt.Printf("  Download Path: %s\n", cfg.DownloadPath)
	}
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
