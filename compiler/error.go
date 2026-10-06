package compiler

import (
	"fmt"
	"strings"

	"ango/engine/script"
)

// Errors is the list of semantic errors found while compiling.
// Compile reports all of them at once, sorted by source position.
type Errors []*script.Error

func (e Errors) Error() string {
	switch len(e) {
	case 0:
		return "no errors"
	case 1:
		return e[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", e[0].Error(), len(e)-1)
}

// Details returns every error on its own line.
func (e Errors) Details() string {
	lines := make([]string, len(e))
	for i, err := range e {
		lines[i] = err.Error()
	}
	return strings.Join(lines, "\n")
}
