package env

import (
	"os"
	"strings"
)

func GetLinuxShell() string {
	shell := os.Getenv("SHELL")
	parts := strings.Split(shell, "/")
	shell = parts[len(parts)-1]
	return shell
}
