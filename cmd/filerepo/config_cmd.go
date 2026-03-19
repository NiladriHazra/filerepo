package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/NiladriHazra/filerepo/internal/config"
)

type configCommand struct {
	scope  string
	action string
	target string
	value  string
	extra  string
}

func parseConfigArgs(args []string) (configCommand, error) {
	if len(args) == 0 {
		return configCommand{}, fmt.Errorf("usage: filerepo config [list|set|unset|profile|favorite|recent]")
	}

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return configCommand{}, fmt.Errorf("usage: filerepo config list")
		}
		return configCommand{scope: "config", action: "list"}, nil
	case "set":
		if len(args) != 3 {
			return configCommand{}, fmt.Errorf("usage: filerepo config set [token|path] VALUE")
		}
		switch args[1] {
		case "token", "path":
			return configCommand{scope: "config", action: "set", target: args[1], value: args[2]}, nil
		default:
			return configCommand{}, fmt.Errorf("unknown config target: %s", args[1])
		}
	case "unset":
		if len(args) != 2 {
			return configCommand{}, fmt.Errorf("usage: filerepo config unset [token|path]")
		}
		switch args[1] {
		case "token", "path":
			return configCommand{scope: "config", action: "unset", target: args[1]}, nil
		default:
			return configCommand{}, fmt.Errorf("unknown config target: %s", args[1])
		}
	case "profile":
		return parseProfileArgs(args[1:])
	case "favorite", "favorites":
		return parseFavoriteArgs(args[1:])
	case "recent":
		return parseRecentArgs(args[1:])
	default:
		return configCommand{}, fmt.Errorf("unknown config action: %s", args[0])
	}
}

func parseProfileArgs(args []string) (configCommand, error) {
	if len(args) == 0 {
		return configCommand{}, fmt.Errorf("usage: filerepo config profile [list|use|set-token|unset-token|import-gh]")
	}

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return configCommand{}, fmt.Errorf("usage: filerepo config profile list")
		}
		return configCommand{scope: "profile", action: "list"}, nil
	case "use":
		if len(args) != 2 {
			return configCommand{}, fmt.Errorf("usage: filerepo config profile use NAME")
		}
		return configCommand{scope: "profile", action: "use", target: args[1]}, nil
	case "set-token":
		if len(args) != 3 {
			return configCommand{}, fmt.Errorf("usage: filerepo config profile set-token NAME TOKEN")
		}
		return configCommand{scope: "profile", action: "set-token", target: args[1], value: args[2]}, nil
	case "unset-token":
		if len(args) != 2 {
			return configCommand{}, fmt.Errorf("usage: filerepo config profile unset-token NAME")
		}
		return configCommand{scope: "profile", action: "unset-token", target: args[1]}, nil
	case "import-gh":
		if len(args) > 2 {
			return configCommand{}, fmt.Errorf("usage: filerepo config profile import-gh [NAME]")
		}
		name := config.DefaultProfileName
		if len(args) == 2 {
			name = args[1]
		}
		return configCommand{scope: "profile", action: "import-gh", target: name}, nil
	default:
		return configCommand{}, fmt.Errorf("unknown profile action: %s", args[0])
	}
}

func parseFavoriteArgs(args []string) (configCommand, error) {
	if len(args) == 0 {
		return configCommand{}, fmt.Errorf("usage: filerepo config favorite [list|add|remove]")
	}

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return configCommand{}, fmt.Errorf("usage: filerepo config favorite list")
		}
		return configCommand{scope: "favorite", action: "list"}, nil
	case "add", "remove":
		if len(args) != 2 {
			return configCommand{}, fmt.Errorf("usage: filerepo config favorite %s URL", args[0])
		}
		return configCommand{scope: "favorite", action: args[0], value: args[1]}, nil
	default:
		return configCommand{}, fmt.Errorf("unknown favorite action: %s", args[0])
	}
}

func parseRecentArgs(args []string) (configCommand, error) {
	if len(args) == 0 {
		return configCommand{}, fmt.Errorf("usage: filerepo config recent [list|clear]")
	}

	switch args[0] {
	case "list", "clear":
		if len(args) != 1 {
			return configCommand{}, fmt.Errorf("usage: filerepo config recent %s", args[0])
		}
		return configCommand{scope: "recent", action: args[0]}, nil
	default:
		return configCommand{}, fmt.Errorf("unknown recent action: %s", args[0])
	}
}

func handleConfig(command configCommand) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch command.scope {
	case "config":
		return handleCoreConfig(cfg, command)
	case "profile":
		return handleProfileConfig(cfg, command)
	case "favorite":
		return handleFavoriteConfig(cfg, command)
	case "recent":
		return handleRecentConfig(cfg, command)
	default:
		return fmt.Errorf("unsupported config scope: %s", command.scope)
	}
}

func handleCoreConfig(cfg config.Config, command configCommand) error {
	switch command.action {
	case "list":
		printConfig(cfg)
		return nil
	case "set":
		switch command.target {
		case "token":
			if err := cfg.SetProfileToken(cfg.ActiveProfileName(), command.value); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("[+] GitHub token saved to profile %q.\n", cfg.ActiveProfileName())
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
		}
	case "unset":
		switch command.target {
		case "token":
			if err := cfg.UnsetProfileToken(cfg.ActiveProfileName()); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("[+] GitHub token removed from profile %q.\n", cfg.ActiveProfileName())
			return nil
		case "path":
			cfg.DownloadPath = ""
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("[+] Download path removed.")
			return nil
		}
	}

	return fmt.Errorf("unsupported config command: %s %s", command.action, command.target)
}

func handleProfileConfig(cfg config.Config, command configCommand) error {
	switch command.action {
	case "list":
		printProfiles(cfg)
		return nil
	case "use":
		if err := cfg.SetActiveProfile(command.target); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("[+] Active profile set to %q.\n", cfg.ActiveProfileName())
		return nil
	case "set-token":
		if err := cfg.SetProfileToken(command.target, command.value); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("[+] Token saved to profile %q.\n", strings.ToLower(command.target))
		return nil
	case "unset-token":
		if err := cfg.UnsetProfileToken(command.target); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("[+] Token removed from profile %q.\n", strings.ToLower(command.target))
		return nil
	case "import-gh":
		token, err := readGitHubCLIToken()
		if err != nil {
			return err
		}
		if err := cfg.SetProfileToken(command.target, token); err != nil {
			return err
		}
		if err := cfg.SetActiveProfile(command.target); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("[+] Imported token from gh into profile %q.\n", cfg.ActiveProfileName())
		return nil
	default:
		return fmt.Errorf("unsupported profile action: %s", command.action)
	}
}

func handleFavoriteConfig(cfg config.Config, command configCommand) error {
	switch command.action {
	case "list":
		printEntries("favorites", cfg.Favorites)
		return nil
	case "add":
		cfg.AddFavorite(command.value)
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("[+] Favorite saved.")
		return nil
	case "remove":
		if !cfg.RemoveFavorite(command.value) {
			return fmt.Errorf("favorite not found: %s", command.value)
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("[+] Favorite removed.")
		return nil
	default:
		return fmt.Errorf("unsupported favorite action: %s", command.action)
	}
}

func handleRecentConfig(cfg config.Config, command configCommand) error {
	switch command.action {
	case "list":
		printEntries("recent repos", cfg.RecentRepos)
		return nil
	case "clear":
		cfg.ClearRecentRepos()
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("[+] Recent repositories cleared.")
		return nil
	default:
		return fmt.Errorf("unsupported recent action: %s", command.action)
	}
}

func printConfig(cfg config.Config) {
	fmt.Println("--- filerepo config ---")
	fmt.Printf("  Active Profile: %s\n", cfg.ActiveProfileName())
	switch token := cfg.ActiveToken(); token {
	case "":
		fmt.Println("  Token:          (not set)")
	default:
		fmt.Printf("  Token:          %s\n", maskToken(token))
	}

	switch cfg.DownloadPath {
	case "":
		fmt.Println("  Download Path:  (default current working directory)")
	default:
		fmt.Printf("  Download Path:  %s\n", cfg.DownloadPath)
	}

	fmt.Printf("  Profiles:       %d\n", len(cfg.ProfileNames()))
	fmt.Printf("  Favorites:      %d\n", len(cfg.Favorites))
	fmt.Printf("  Recent Repos:   %d\n", len(cfg.RecentRepos))
	fmt.Printf("  Cache:          enabled=%t ttl=%s\n", cfg.Cache.Enabled, cfg.CacheTTL())
}

func printProfiles(cfg config.Config) {
	fmt.Println("--- filerepo profiles ---")
	for _, name := range cfg.ProfileNames() {
		prefix := " "
		if name == cfg.ActiveProfileName() {
			prefix = "*"
		}
		token := cfg.Profiles[name].GitHubToken
		if token == "" {
			fmt.Printf("%s %s  (no token)\n", prefix, name)
			continue
		}
		fmt.Printf("%s %s  %s\n", prefix, name, maskToken(token))
	}
}

func printEntries(title string, entries []config.RepoEntry) {
	fmt.Printf("--- filerepo %s ---\n", title)
	if len(entries) == 0 {
		fmt.Println("  (empty)")
		return
	}

	for index, entry := range entries {
		if entry.LastUsedAt.IsZero() {
			fmt.Printf("  %d. %s\n", index+1, entry.URL)
			continue
		}
		fmt.Printf("  %d. %s  [%s]\n", index+1, entry.URL, entry.LastUsedAt.Local().Format("2006-01-02 15:04"))
	}
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func readGitHubCLIToken() (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("gh", "auth", "token")
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		switch {
		case stderr.Len() > 0:
			return "", fmt.Errorf("read token from gh: %s", strings.TrimSpace(stderr.String()))
		default:
			return "", fmt.Errorf("read token from gh: %w", err)
		}
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("gh auth token returned an empty value")
	}
	return token, nil
}
