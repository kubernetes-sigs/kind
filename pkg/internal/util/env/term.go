package env

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// IsTerminal returns true if the writer w is a terminal
func IsTerminal(w io.Writer) bool {
	if v, ok := (w).(*os.File); ok {
		return terminal.IsTerminal(int(v.Fd()))
	}
	return false
}
