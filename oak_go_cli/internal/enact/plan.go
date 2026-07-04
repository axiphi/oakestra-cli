package enact

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/oakestra/oak-go-cli/internal/cliout"
)

// Plan holds a series of steps to achieve a specific goal.
type Plan struct {
	Info  string
	Goal  string
	Steps []Step
}

// Execute asks for confirmation (if needed) and runs the steps sequentially.
func (p *Plan) Execute(autoConfirm bool) (bool, error) {
	var stepDescriptions string
	for _, step := range p.Steps {
		stepDescriptions += ">> " + step.Description() + "\n"
	}

	if autoConfirm {
		cliout.Infof("%s, running these steps to %s:\n%s", p.Info, p.Goal, stepDescriptions)
	} else {
		var confirmed = true
		err := huh.NewConfirm().
			Title(fmt.Sprintf("%s,\nrun these steps to %s?\n%s", p.Info, p.Goal, stepDescriptions)).
			Value(&confirmed).
			Run()

		if err != nil {
			return false, err
		}

		if !confirmed {
			return false, nil
		}
	}

	for _, step := range p.Steps {
		if err := step.Run(); err != nil {
			return true, fmt.Errorf("failed during step [%s]: %w", step.Description(), err)
		}
	}

	return true, nil
}
