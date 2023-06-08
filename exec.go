package puffin

import (
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// OsExec is an Exec implementation that uses os/exec functions
type OsExec struct{}

// NewOsExec creates a new OsExec struct
func NewOsExec() Exec {
	return &OsExec{}
}

// LookPath behaves the same as exec.LookPath https://pkg.go.dev/os/exec#LookPath
func (*OsExec) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Command behaves the same as exec.Command https://pkg.go.dev/os/exec#Command
func (*OsExec) Command(name string, arg ...string) Cmd {
	return &OsCmd{Cmd: exec.Command(name, arg...)}
}

// CommandContext behaves the same as exec.CommandContext https://pkg.go.dev/os/exec#CommandContext
func (*OsExec) CommandContext(ctx context.Context, name string, arg ...string) Cmd {
	return &OsCmd{Cmd: exec.CommandContext(ctx, name, arg...)}
}

type OsCmd struct {
	*exec.Cmd
}

// Path returns the Cmd path https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Path() string {
	return c.Cmd.Path
}

// SetPath sets the Cmd path https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SetPath(path string) {
	c.Cmd.Path = path
}

// Args returns the Cmd args https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Args() []string {
	return c.Cmd.Args
}

// SetArgs sets the Cmd args https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SetArgs(args []string) {
	c.Cmd.Args = args
}

// Env returns the Cmd env https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Env() []string {
	return c.Cmd.Env
}

// SetEnv sets the Cmd env https://pkg.go.dev/os/exec#Cmd
// Replaces all existing envs with this slice; does not append.
func (c *OsCmd) SetEnv(env []string) {
	c.Cmd.Env = env
}

// Dir returns the Cmd working dir https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Dir() string {
	return c.Cmd.Dir
}

// SetDir sets the Cmd working dir https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SetDir(dir string) {
	c.Cmd.Dir = dir
}

// Stdin returns the Cmd Stdin https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Stdin() io.Reader {
	return c.Cmd.Stdin
}

// SetStdin sets the Cmd Stdin https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SetStdin(stdin io.Reader) {
	c.Cmd.Stdin = stdin
}

// Stdout returns the Cmd Stdout https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Stdout() io.Writer {
	return c.Cmd.Stdout
}

// SetStdout sets the Cmd Stdout https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SetStdout(stdout io.Writer) {
	c.Cmd.Stdout = stdout
}

// Stderr returns the Cmd Stderr https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Stderr() io.Writer {
	return c.Cmd.Stderr
}

// SetStderr sets the Cmd Stderr https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SetStderr(stderr io.Writer) {
	c.Cmd.Stderr = stderr
}

// ExtraFiles returns the Cmd extra files https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) ExtraFiles() []*os.File {
	return c.Cmd.ExtraFiles
}

// SetExtraFiles sets the Cmd extra files https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SetExtraFiles(extraFiles []*os.File) {
	c.Cmd.ExtraFiles = extraFiles
}

// SysProcAttr returns the Cmd sys proc attr https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) SysProcAttr() *syscall.SysProcAttr {
	return c.Cmd.SysProcAttr
}

// Process returns the Cmd process https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Process() *os.Process {
	return c.Cmd.Process
}

// ProcessState returns the Cmd process state https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) ProcessState() *os.ProcessState {
	return c.Cmd.ProcessState
}

// Err returns the Cmd err https://pkg.go.dev/os/exec#Cmd
func (c *OsCmd) Err() error {
	return c.Cmd.Err
}
