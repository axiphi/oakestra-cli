package cliout

import (
	"os"

	"golang.org/x/term"
)

// colorsEnabled is true when stdout is an interactive terminal and NO_COLOR is unset.
var colorsEnabled = term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == ""

func ansi(code, s string) string {
	if !colorsEnabled {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func Bold(s string) string { return ansi("1", s) }
func Dim(s string) string  { return ansi("2", s) }

func Green(s string) string  { return ansi("32", s) }
func Yellow(s string) string { return ansi("33", s) }
func Cyan(s string) string   { return ansi("36", s) }
func Red(s string) string    { return ansi("31", s) }
