package enact

import (
	"strings"

	"github.com/oakestra/oak-go-cli/internal/cmd"
)

// Step represents a single mutation to the system.
type Step interface {
	// Description returns a human-readable explanation of what the step does,
	// used for the confirmation prompt.
	Description() string

	// Run executes the mutation.
	Run() error
}

// CommandStep runs an external system command.
type CommandStep struct {
	Command string
	Args    []string
}

func (c CommandStep) Description() string {
	return "Run: " + c.Command + " " + strings.Join(c.Args, " ")
}

func (c CommandStep) Run() error {
	// Replaces your existing runCmd call
	return cmd.RunForwarded(c.Command, c.Args...)
}

// FuncStep executes a custom Go function (lambda).
type FuncStep struct {
	Name string // A description of what the function does
	Fn   func() error
}

func (f FuncStep) Description() string {
	return f.Name
}

func (f FuncStep) Run() error {
	return f.Fn()
}
