package pat

import (
	"path/filepath"

	"golang.org/x/exp/slices"

	"github.com/weave-lab/puffin"
)

// Pattern is a simple pattern mather that matches a command based on a
// command name and it's arguments
type Pattern struct {
	Cmd  string
	Args []string
}

// New creates a new Pattern matcher
// a cmd of "*" will match all commands
func New(cmd string, args ...string) *Pattern {
	return &Pattern{
		Cmd:  cmd,
		Args: args,
	}
}

// All creates a new Pattern matcher that matches all commands
func All() *Pattern {
	return &Pattern{
		Cmd: "*",
	}
}

// Match matches a FuncCmd if it has a matching command name
// and matching arguments. The order of the arguments is not
// taken into account when matching
func (p *Pattern) Match(cmd *puffin.FuncCmd) bool {
	if p.Cmd != "*" && filepath.Base(p.Cmd) != cmd.Path() {
		return false
	}

	for _, arg := range p.Args {
		if !slices.Contains(cmd.Args()[1:], arg) {
			return false
		}
	}

	return true
}
