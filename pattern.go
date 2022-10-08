package puffin

import "path/filepath"

// pat is a simple Pattern that matches a command by its name
// it is needed because importing puffin/pat causes an import cycle
// a pat of "*" will match all command names
type pat string

// Match implements the Pattern interface
func (p pat) Match(cmd *FuncCmd) bool {
	if p != "*" && filepath.Base(string(p)) != cmd.Path() {
		return false
	}

	return true
}
