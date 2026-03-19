package ui

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/muesli/termenv"
)

func copyText(value string) error {
	if stringsTrimmed(value) == "" {
		return fmt.Errorf("nothing to copy")
	}

	if err := copyWithSystemClipboard(value); err == nil {
		return nil
	}

	termenv.Copy(value)
	return nil
}

func copyWithSystemClipboard(value string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command(path)
		} else if path, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command(path, "-selection", "clipboard")
		} else {
			return fmt.Errorf("no clipboard command available")
		}
	}

	cmd.Stdin = bytes.NewBufferString(value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copy to clipboard: %w", err)
	}
	return nil
}
