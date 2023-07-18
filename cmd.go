package puffin

import (
	"io"
	"os"
	"syscall"
)

// Cmd is the interface for a command runner
type Cmd interface {
	// CombinedOutput runs the command and returns its
	// combined standard output and standard error.
	CombinedOutput() ([]byte, error)

	// Environ returns a copy of the environment in which
	// the command would be run as it is currently configured.
	Environ() []string

	// Output runs the command and returns its standard output.
	Output() ([]byte, error)

	// Run starts the specified command and waits for it to complete.
	Run() error

	// Start starts the specified command but does not wait for it to complete.
	Start() error

	// StderrPipe returns a pipe that will be connected to the command's
	// standard error when the command starts.
	StderrPipe() (io.ReadCloser, error)

	// StdinPipe returns a pipe that will be connected to the command's
	// standard input when the command starts.
	StdinPipe() (io.WriteCloser, error)

	// StdoutPipe returns a pipe that will be connected to the command's
	// standard output when the command starts.
	StdoutPipe() (io.ReadCloser, error)

	// String returns a human-readable description of Cmd
	String() string

	// Wait waits for the command to exit and waits for any
	// copying to stdin, stdout, or stderr to complete.
	Wait() error

	// Path returns the path of the command to run.
	Path() string

	// SetPath sets the path of the command to run.
	SetPath(string)

	// Args returns the command line arguments, including the command as Args[0]
	Args() []string

	// SetArgs sets the command line arguments
	SetArgs([]string)

	// Env returns the environment of the process
	Env() []string

	// SetEnv sets the environment of the process
	SetEnv([]string)

	// Dir returns the working directory of the command
	Dir() string

	// SetDir sets the working dir of the command
	SetDir(string)

	// Stdin returns the command's standard input
	Stdin() io.Reader

	// SetStdin sets the command's standard input
	SetStdin(io.Reader)

	// Stdout returns the command's standard output
	Stdout() io.Writer

	// SetStdout sets the command's standard output
	SetStdout(io.Writer)

	// Stderr returns the command's standard error
	Stderr() io.Writer

	// SetStderr sets the command's standard error
	SetStderr(io.Writer)

	// ExtraFiles returns additional open files to be inherited by the new process
	ExtraFiles() []*os.File

	// SetExtraFiles set additional open files to be inherited by the new process
	SetExtraFiles([]*os.File)

	// SysProcAttr returns optional, operating system-specific attributes
	SysProcAttr() *syscall.SysProcAttr

	// SetSysProcAttr sets the SysProcAttr field on the underlying command
	SetSysProcAttr(*syscall.SysProcAttr)

	// Process is the underlying process, once started.
	Process() *os.Process

	// ProcessState contains information about an exited process,
	// available after a call to Wait or Run.
	ProcessState() *os.ProcessState

	// Err contains a LookPath error, if any
	Err() error
}
