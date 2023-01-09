package puffin

// Pattern is a pattern matcher for a FuncCmd
type Pattern interface {
	Match(*FuncCmd) bool
}

// muxMatcher maps a pat.Pattern to a specific CmdFunc handler
type muxMatcher struct {
	pat     Pattern
	handler CmdFunc
}

// Mux routs incoming commands to the appropriate handler functions
type Mux struct {
	matchers []muxMatcher
}

// NewMux returns a new mux with no configured handlers
func NewMux() *Mux {
	return &Mux{}
}

// NewHandlerMux returns a new mux with a single handler that matches all commands
func NewHandlerMux(handler CmdFunc) *Mux {
	return &Mux{
		matchers: []muxMatcher{{
			pat:     pat("*"),
			handler: handler,
		}},
	}
}

// NewFuncMapMux adds new handlers to the mux for each of the functions in the func map.
// The keys are used to match commands by their names.
func NewFuncMapMux(funcMap map[string]CmdFunc) *Mux {
	mux := &Mux{}

	var addLast CmdFunc
	for key, fn := range funcMap {
		// make sure the catch-all case is the final matcher
		if key == "*" {
			addLast = fn
			continue
		}

		mux.HandleFunc(pat(key), fn)
	}

	if addLast != nil {
		mux.HandleFunc(pat("*"), addLast)
	}

	return mux
}

// NewStdoutMux returns a new mux with a single handler that matches all commands
// it will write the provided message to stdout and return a 0 error code
func NewStdoutMux(stdout string) *Mux {
	return &Mux{
		matchers: []muxMatcher{{
			pat: pat("*"),
			handler: func(cmd *FuncCmd) int {
				_, err := cmd.Stdout().Write([]byte(stdout))
				if err != nil {
					// return a non-zero error code since the stdout write failed
					return 1
				}

				return 0
			},
		}},
	}
}

// NewStderrMux returns a new mux with a single handler that matches all commands
// it will write the provided message to stderr and return the provided error code
func NewStderrMux(stderr string, errCode int) *Mux {
	return &Mux{
		matchers: []muxMatcher{{
			pat: pat("*"),
			handler: func(cmd *FuncCmd) int {
				// this failure should be really rare, so we can probably just ignore it
				_, _ = cmd.Stderr().Write([]byte(stderr))

				return errCode
			},
		}},
	}
}

// HandleFunc adds a new handler func to the mux with the given Pattern matcher
func (m *Mux) HandleFunc(pat Pattern, handler CmdFunc) {
	m.matchers = append(m.matchers, muxMatcher{
		pat:     pat,
		handler: handler,
	})
}

// findHandler searches through all the mux's Patterns to find a match
// if a match is found the corresponding CmdFunc is also returned
func (m *Mux) findHandler(cmd *FuncCmd) CmdFunc {
	if m == nil {
		return nil
	}

	for _, matcher := range m.matchers {
		if matcher.pat.Match(cmd) {
			return matcher.handler
		}
	}

	return nil
}
