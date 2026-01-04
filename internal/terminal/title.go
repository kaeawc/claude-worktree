// Package terminal provides helpers for terminal-specific behavior.
package terminal

import (
	"fmt"
	"os"
)

// SetTitle sets the terminal window title using an ANSI escape sequence.
func SetTitle(title string) {
	if title == "" {
		return
	}

	//nolint:errcheck
	_, _ = fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
}
