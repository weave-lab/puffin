package puffin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"golang.org/x/exp/slices"
)

// PidMax is the default max on most linux OS
const PidMax = 32768

// FuncExec is an Exec implementation that uses provided go functions
// rather than the os/exec package
type FuncExec struct {
	mux  *Mux
	bins []string
	envs map[string]string
}

// NewFuncExec creates a new FuncExec struct
func NewFuncExec(mux *Mux, opts ...FuncExecOption) Exec {
	fExec := &FuncExec{
		mux: mux,
	}

	for _, opt := range opts {
		opt(fExec)
	}

	// if no bins are specified add a wildcard matcher
	// so everything is found by default
	if len(fExec.bins) == 0 {
		fExec.bins = []string{"*"}
	}

	return fExec
}

// LookPath always returns the file unless bins is set.
// if bins is set file will be matched against the provided bins
// if the file contains a / then it must match a bin exactly
// otherwise bins are matched to files based on the bins filepath base
func (e *FuncExec) LookPath(file string) (string, error) {
	hasWildcard := slices.Contains(e.bins, "*")

	if strings.Contains(file, "/") {
		if slices.Contains(e.bins, file) {
			return file, nil
		}

		// if nothing else matches but bins has * matcher still return a match
		if hasWildcard {
			return file, nil
		}

		// must be an exact match if there's a / in the path name
		return "", &exec.Error{Name: file, Err: &os.PathError{Op: "stat", Path: file, Err: syscall.ENOENT}}
	}

	// otherwise we can just match the name with the filepath base
	for _, p := range e.bins {
		if filepath.Base(p) == file {
			return p, nil
		}
	}

	// if nothing else matches but bins has * matcher still return a match
	if hasWildcard {
		return file, nil
	}

	return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
}

// Command creates a new Cmd that uses execs funcMap rather than the shell for
// running commands
func (e *FuncExec) Command(name string, arg ...string) Cmd {
	cmd := &FuncCmd{
		path:     name,
		args:     append([]string{name}, arg...),
		ctxErr:   make(chan error, 1),
		exitCode: make(chan int, 1),
		fExec:    e,
	}
	if filepath.Base(name) == name {
		lp, err := e.LookPath(name)
		if lp != "" {
			cmd.path = lp
		}
		if err != nil {
			cmd.err = err
		}
	}

	return cmd
}

// CommandContext works the same as Command except it includes a context that
// can be used to cancle the commands execution
func (e *FuncExec) CommandContext(ctx context.Context, name string, arg ...string) Cmd {
	if ctx == nil {
		panic("nil Context")
	}
	cmd := e.Command(name, arg...)
	cmd.(*FuncCmd).ctx = ctx

	return cmd
}

// FuncExecOption can be used to configure the FuncExec struct
type FuncExecOption func(*FuncExec)

// WithBins adds the specified bins to the pseudo $PATH used by this Exec
// if this is not included all command paths will be 'found' by the Exec
func WithBins(bins ...string) FuncExecOption {
	return func(fExec *FuncExec) {
		fExec.bins = append(fExec.bins, bins...)
	}
}

// WithEnv sets the env used by all the commands created by this Exec
func WithEnv(envs map[string]string) FuncExecOption {
	return func(fExec *FuncExec) {
		fExec.envs = envs
	}
}

// CmdFunc is a function that will run based on the command name
type CmdFunc func(*FuncCmd) int

// FuncCmd is a Cmd that runs a CmdFnd rather than running in a shell
type FuncCmd struct {
	path string
	args []string
	env  map[string]string
	dir  string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	extraFiles   []*os.File
	sysProcAttr  *syscall.SysProcAttr
	process      *os.Process
	processState *os.ProcessState

	ctx      context.Context
	ctxErr   chan error
	err      error
	startErr error
	exitCode chan int

	fExec *FuncExec
}

// CombinedOutput runs the command function and returns its combined standard
// output and standard error.
func (c *FuncCmd) CombinedOutput() ([]byte, error) {
	if c.stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	b := newLockableBuffer()
	c.stdout = b
	c.stderr = b
	err := c.Run()
	return b.Bytes(), err
}

// Environ returns a copy of the environment in which the command function would be run
// as it is currently configured.
func (c *FuncCmd) Environ() []string {
	env := c.env
	if env == nil && c.fExec != nil {
		return fmtEnv(c.fExec.envs)
	}

	return fmtEnv(c.env)
}

// fmtEnv converts a map of env names and values into a slice of strings
func fmtEnv(env map[string]string) []string {
	var fmted []string
	for k, v := range env {
		fmted = append(fmted, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(fmted)

	return fmted
}

// Output runs the command function and returns its standard output.
func (c *FuncCmd) Output() ([]byte, error) {
	if c.stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	stdout := newLockableBuffer()
	c.stdout = stdout

	captureErr := c.stderr == nil
	if captureErr {
		// TODO: port prefixSuffixSaver so error messages are the same as the os/exec package errors
		c.stderr = newLockableBuffer()
	}

	err := c.Run()
	if err != nil && captureErr {
		if ee, ok := err.(*exec.ExitError); ok {
			ee.Stderr = c.stderr.(*lockableBuffer).Bytes()
		}
	}
	return stdout.Bytes(), err
}

// Run starts the specified command and waits for it to complete.
func (c *FuncCmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Start starts the specified command and returns imediately.
// Wait should be called to wait for the command to complete.
func (c *FuncCmd) Start() error {
	if c.path == "" && c.err == nil {
		c.err = errors.New("exec: no command")
	}
	if c.err != nil {
		return c.err
	}
	if c.Process() != nil {
		return errors.New("exec: already started")
	}
	if c.exitCode == nil {
		return errors.New("cmd missing exitCode channel")
	}
	if c.ctx != nil && c.ctxErr == nil {
		return errors.New("cmd missing ctxErr channel")
	}

	// check if the context is already done
	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}
	}

	c.process = &os.Process{Pid: rand.Intn(PidMax)}

	fn := c.fExec.mux.findHandler(c)
	if fn == nil {
		c.startErr = errors.New("exit status 1")
		return nil
	}

	done := make(chan struct{})
	// start the command function in a go routine
	go func() {
		c.exitCode <- fn(c)
		done <- struct{}{}
	}()

	// listen for the command to be canceled if ctx is not nil
	if c.ctx != nil {
		go func() {
			select {
			case <-c.ctx.Done():
				c.lock()
				c.ctxErr <- c.ctx.Err()
			case <-done:
				c.ctxErr <- nil
			}
		}()
	}

	return nil
}

// StderrPipe returns a io.ReadCloser that is attached to the cmds Stderr
func (c *FuncCmd) StderrPipe() (io.ReadCloser, error) {
	if c.stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	if c.process != nil {
		return nil, errors.New("exec: StderrPipe after process started")
	}

	buf := newLockableBuffer()
	c.stderr = buf
	return buf, nil
}

// StdinPipe returns an io.WriteCloser that is attached to the cmds Stdin
func (c *FuncCmd) StdinPipe() (io.WriteCloser, error) {
	if c.stdin != nil {
		return nil, errors.New("exec: Stdin already set")
	}
	if c.process != nil {
		return nil, errors.New("exec: StdinPipe after process started")
	}

	buf := newLockableBuffer()
	c.stdin = buf
	return buf, nil
}

// StdoutPipe returns an io.ReadCloser that is attached to  to the cmds Stdout
func (c *FuncCmd) StdoutPipe() (io.ReadCloser, error) {
	if c.stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.process != nil {
		return nil, errors.New("exec: StdoutPipe after process started")
	}

	buf := newLockableBuffer()
	c.stdout = buf
	return buf, nil
}

// String returns a human-readable description of the cmd
func (c *FuncCmd) String() string {
	if c.err != nil {
		// failed to resolve path; report the original requested path (plus args)
		return strings.Join(c.args, " ")
	}
	// report the exact executable path (plus args)
	var b strings.Builder
	b.WriteString(c.path)
	for _, a := range c.args[1:] {
		b.WriteByte(' ')
		b.WriteString(a)
	}
	return b.String()
}

func (c *FuncCmd) Wait() error {
	if c.startErr != nil {
		return c.startErr
	}

	if c.process == nil {
		return errors.New("exec: not started")
	}
	if c.processState != nil {
		return errors.New("exec: Wait was already called")
	}
	c.processState = &os.ProcessState{}

	if c.ctx != nil {
		interruptErr := <-c.ctxErr
		if interruptErr != nil {
			return interruptErr
		}
	}

	exitCode := <-c.exitCode
	if exitCode != 0 {
		return fmt.Errorf("exit status %d", exitCode)
	}

	return nil
}

// lock, prevents further changes to the underlying commands buffers
func (c *FuncCmd) lock() {
	if r, ok := c.stdin.(*lockableBuffer); ok {
		r.LockRead()
	}

	if w, ok := c.stdout.(*lockableBuffer); ok {
		w.LockWrite()
	}

	if w, ok := c.stderr.(*lockableBuffer); ok {
		w.LockWrite()
	}
}

// Path returns the Cmd path
func (c *FuncCmd) Path() string {
	return c.path
}

// SetPath sets the Cmd path
func (c *FuncCmd) SetPath(path string) {
	c.path = path
}

// Args returns the Cmd args
func (c *FuncCmd) Args() []string {
	return c.args
}

// SetArgs sets the Cmd args
func (c *FuncCmd) SetArgs(args []string) {
	c.args = args
}

// Env returns the Cmd env
func (c *FuncCmd) Env() []string {
	return fmtEnv(c.env)
}

// SetEnv sets the Cmd env
func (c *FuncCmd) SetEnv(env []string) {
	if len(env) == 0 {
		return
	}

	if c.env == nil {
		c.env = make(map[string]string, len(env))
	}

	for _, e := range env {
		nameVal := strings.SplitN(e, "=", 2)
		name := nameVal[0]
		var val string
		if len(nameVal) >= 2 {
			val = nameVal[1]
		}
		c.env[name] = val
	}
}

// Dir returns the Cmd working dir
func (c *FuncCmd) Dir() string {
	return c.dir
}

// SetDir sets the Cmd working dir
func (c *FuncCmd) SetDir(dir string) {
	c.dir = dir
}

// Stdin returns the Cmd Stdin
func (c *FuncCmd) Stdin() io.Reader {
	return c.stdin
}

// SetStdin sets the Cmd Stdin
func (c *FuncCmd) SetStdin(stdin io.Reader) {
	c.stdin = newLockableReader(stdin)
}

// Stdout returns the Cmd Stdout
func (c *FuncCmd) Stdout() io.Writer {
	if writer, ok := c.stdout.(*lockableBuffer); ok {
		return writer.writer
	}

	return c.stdout
}

// SetStdout sets the Cmd Stdout
func (c *FuncCmd) SetStdout(stdout io.Writer) {
	c.stdout = newLockableWriter(stdout)
}

// Stderr returns the Cmd Stderr
func (c *FuncCmd) Stderr() io.Writer {
	if writer, ok := c.stderr.(*lockableBuffer); ok {
		return writer.writer
	}

	return c.stderr
}

// SetStderr sets the Cmd Stderr
func (c *FuncCmd) SetStderr(stderr io.Writer) {
	c.stderr = newLockableWriter(stderr)
}

// ExtraFiles returns the Cmd extra files
func (c *FuncCmd) ExtraFiles() []*os.File {
	return c.extraFiles
}

// SetExtraFiles sets the Cmd extra files
func (c *FuncCmd) SetExtraFiles(extraFiles []*os.File) {
	c.extraFiles = extraFiles
}

// SysProcAttr returns the Cmd sys proc attr
func (c *FuncCmd) SysProcAttr() *syscall.SysProcAttr {
	return c.sysProcAttr
}

// Process returns the Cmd process
func (c *FuncCmd) Process() *os.Process {
	return c.process
}

// ProcessState returns the Cmd process state
func (c *FuncCmd) ProcessState() *os.ProcessState {
	return c.processState
}

// Err returns the Cmd err
func (c *FuncCmd) Err() error {
	return c.err
}
