package puffin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestExec_LookPath(t *testing.T) {
	type fields struct {
		bins []string
	}
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			"missing func",
			fields{
				bins: []string{"/fail/path/to/test"},
			},
			args{
				file: "/path/to/test",
			},
			"",
			true,
		},
		{
			"found command",
			fields{
				bins: []string{"test"},
			},
			args{
				file: "test",
			},
			"test",
			false,
		},
		{
			"found command file",
			fields{
				bins: []string{"/path/to/test"},
			},
			args{
				file: "test",
			},
			"/path/to/test",
			false,
		},
		{
			"found file",
			fields{
				bins: []string{"/path/to/test"},
			},
			args{
				file: "/path/to/test",
			},
			"/path/to/test",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				bins: tt.fields.bins,
			}
			got, err := e.LookPath(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FuncExec.LookPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("FuncExec.LookPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuncExec_Command(t *testing.T) {
	type fields struct {
		mux  *Mux
		bins []string
		envs map[string]string
	}
	type args struct {
		name string
		arg  []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *FuncCmd
	}{
		{
			"cmd with args",
			fields{
				mux: &Mux{
					matchers: []muxMatcher{
						{pat: pat("test")},
					},
				},
				bins: []string{"*"},
			},
			args{
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"different path",
			fields{
				mux: &Mux{
					matchers: []muxMatcher{
						{pat: pat("/path/to/test")},
					},
				},
				bins: []string{"/path/to/test"},
			},
			args{
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				path:     "/path/to/test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"lookpath failure",
			fields{
				bins: []string{"missing"},
			},
			args{
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				err:      errors.New(`exec: "test": executable file not found in $PATH`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				bins: tt.fields.bins,
				envs: tt.fields.envs,
			}

			// set up tt.want with the correct fExec
			tt.want.fExec = e

			got := e.Command(tt.args.name, tt.args.arg...)

			// test err filrst since errors don't play nice with DeepEqual
			var gotErr, wantErr string
			if got.Err() != nil {
				gotErr = got.Err().Error()
				got.(*FuncCmd).err = nil
			}
			if tt.want.Err() != nil {
				wantErr = tt.want.Err().Error()
				tt.want.err = nil
			}
			if gotErr != wantErr {
				t.Errorf("FuncExec.Command() got err %s, wanted err %s", gotErr, wantErr)
			}

			// check channels seperately as well
			if len(got.(*FuncCmd).exitCode) != len(tt.want.exitCode) ||
				cap(got.(*FuncCmd).exitCode) != cap(tt.want.exitCode) {
				t.Errorf("FuncExec.CommandContext() exitCode chan did not match what was expected")
			}
			if len(got.(*FuncCmd).ctxErr) != len(tt.want.ctxErr) ||
				cap(got.(*FuncCmd).ctxErr) != cap(tt.want.ctxErr) {
				t.Errorf("FuncExec.CommandContext() ctxErr chan did not match what was expected")
			}
			got.(*FuncCmd).exitCode = nil
			got.(*FuncCmd).ctxErr = nil
			tt.want.exitCode = nil
			tt.want.ctxErr = nil

			// now test the rest of the Cmd
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncExec.Command() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuncExec_CommandContext(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	type fields struct {
		mux  *Mux
		bins []string
		envs map[string]string
	}
	type args struct {
		ctx  context.Context
		name string
		arg  []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *FuncCmd
	}{
		{
			"cmd with args",
			fields{
				mux: &Mux{
					matchers: []muxMatcher{
						{pat: pat("test")},
					},
				},
				bins: []string{"*"},
			},
			args{
				ctx:  context.Background(),
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      context.Background(),
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"different path",
			fields{
				mux: &Mux{
					matchers: []muxMatcher{
						{pat: pat("/path/to/test")},
					},
				},
				bins: []string{"/path/to/test"},
			},
			args{
				ctx:  context.Background(),
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      context.Background(),
				path:     "/path/to/test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
		{
			"lookpath failure",
			fields{
				bins: []string{"/bin/missing"},
			},
			args{
				ctx:  context.Background(),
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      context.Background(),
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				err:      errors.New(`exec: "test": executable file not found in $PATH`),
			},
		},
		{
			"with timeout context",
			fields{
				mux: &Mux{
					matchers: []muxMatcher{
						{pat: pat("test")},
					},
				},
				bins: []string{"*"},
			},
			args{
				ctx:  timeoutCtx,
				name: "test",
				arg:  []string{"arg1", "arg2"},
			},
			&FuncCmd{
				ctx:      timeoutCtx,
				path:     "test",
				args:     []string{"test", "arg1", "arg2"},
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &FuncExec{
				mux:  tt.fields.mux,
				bins: tt.fields.bins,
				envs: tt.fields.envs,
			}

			// set up tt.want with the correct fExec
			tt.want.fExec = e

			got := e.CommandContext(tt.args.ctx, tt.args.name, tt.args.arg...)

			// test err filrst since errors don't play nice with DeepEqual
			var gotErr, wantErr string
			if got.Err() != nil {
				gotErr = got.Err().Error()
				got.(*FuncCmd).err = nil
			}
			if tt.want.Err() != nil {
				wantErr = tt.want.Err().Error()
				tt.want.err = nil
			}
			if gotErr != wantErr {
				t.Errorf("FuncExec.CommandContext() got err %s, wanted err %s", gotErr, wantErr)
			}

			// check channels seperately as well
			if len(got.(*FuncCmd).exitCode) != len(tt.want.exitCode) ||
				cap(got.(*FuncCmd).exitCode) != cap(tt.want.exitCode) {
				t.Errorf("FuncExec.CommandContext() exitCode chan did not match what was expected")
			}
			if len(got.(*FuncCmd).ctxErr) != len(tt.want.ctxErr) ||
				cap(got.(*FuncCmd).ctxErr) != cap(tt.want.ctxErr) {
				t.Errorf("FuncExec.CommandContext() ctxErr chan did not match what was expected")
			}
			got.(*FuncCmd).exitCode = nil
			got.(*FuncCmd).ctxErr = nil
			tt.want.exitCode = nil
			tt.want.ctxErr = nil

			// now test the rest of the Cmd
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncExec.CommandContext() = \n%#v, want \n%#v", got, tt.want)
			}
		})
	}
}

func TestFuncCmd_Start(t *testing.T) {
	type fields struct {
		path     string
		stdout   io.Writer
		process  *os.Process
		exitCode chan int
		ctx      context.Context
		err      error
		fExec    *FuncExec
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty path",
			fields{
				path: "",
			},
			true,
		},
		{
			"lookpath err",
			fields{
				err: errors.New("lookpath error"),
			},
			true,
		},
		{
			"already run",
			fields{
				process: &os.Process{},
			},
			true,
		},
		{
			"context is already done",
			fields{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			true,
		},
		{
			"run cmd",
			fields{
				path:     "test",
				exitCode: make(chan int, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("Start() failed to setup test, %s", err)
									}
									return 0
								},
							},
						},
					},
				},
				stdout: &bytes.Buffer{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				stdout:   tt.fields.stdout,
				process:  tt.fields.process,
				exitCode: tt.fields.exitCode,
				err:      tt.fields.err,
				fExec:    tt.fields.fExec,
			}
			if err := c.Start(); (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// check that process and pid are set
			if c.process == nil {
				t.Fatalf("FuncCmd.Start() start was run but process was not set")
			}

			if c.process.Pid == 0 {
				t.Fatalf("FuncCmd.Start() start was run but process id was not set")
			}

			// sleep for a short time to make sure the cmf func has time to run
			time.Sleep(time.Millisecond)

			// check that the function has been run (or at least started)
			gotStdout := c.stdout.(*bytes.Buffer).String()
			wantStdout := "test was run"
			if gotStdout != wantStdout {
				t.Errorf("FuncCmd.Start() cmd func was not run, got stdout %s but wanted %s", gotStdout, wantStdout)
			}
		})
	}
}

func TestFuncCmd_Wait(t *testing.T) {
	type fields struct {
		process      *os.Process
		processState *os.ProcessState
		ctx          context.Context
		ctxErr       chan error
		startErr     error
		exitCode     chan int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"not started",
			fields{
				process: nil,
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"start error",
			fields{
				startErr: errors.New("start error"),
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"wait already called",
			fields{
				process:      &os.Process{},
				processState: &os.ProcessState{},
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"process failed",
			fields{
				process: &os.Process{},
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 1 // non zero exit code
					return code
				}(),
			},
			true,
		},
		{
			"context already canceled",
			fields{
				process: &os.Process{},
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				ctxErr: func() chan error {
					err := make(chan error, 1)
					err <- errors.New("context canceled")
					return err
				}(),
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			true,
		},
		{
			"success",
			fields{
				process: &os.Process{},
				exitCode: func() chan int {
					code := make(chan int, 1)
					code <- 0
					return code
				}(),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				process:      tt.fields.process,
				processState: tt.fields.processState,
				ctx:          tt.fields.ctx,
				ctxErr:       tt.fields.ctxErr,
				startErr:     tt.fields.startErr,
				exitCode:     tt.fields.exitCode,
			}
			if err := c.Wait(); (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.Wait() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// check that process state was set
			if c.processState == nil {
				t.Errorf("FuncCmd.Wait() process state was not set")
			}
		})
	}
}

func TestFuncCmd_Run(t *testing.T) {
	type fields struct {
		path       string
		stdout     io.Writer
		ctxTimeout time.Duration
		ctxErr     chan error
		exitCode   chan int
		fExec      *FuncExec
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		wantStart bool
	}{
		{
			"canceled command stdout locked",
			fields{
				path:       "slow",
				ctxTimeout: time.Millisecond * 5,
				exitCode:   make(chan int, 1),
				ctxErr:     make(chan error, 1),
				stdout:     newLockableBuffer(),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("slow"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("Run() failed to run slow command %v", err)
									}
									time.Sleep(time.Millisecond * 10)
									_, err = fc.stdout.Write([]byte("sleep finished"))
									if err != nil {
										t.Fatalf("Run() failed to run slow command %v", err)
									}
									return 0
								},
							},
						},
					},
				},
			},
			true,
			true,
		},
		{
			"uncanceled command",
			fields{
				path:       "fast",
				ctxTimeout: time.Minute,
				exitCode:   make(chan int, 1),
				ctxErr:     make(chan error, 1),
				stdout:     newLockableBuffer(),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("fast"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										return 1
									}
									return 0
								},
							},
						},
					},
				},
			},
			false,
			true,
		},
		{
			"command failed",
			fields{
				path:     "test",
				exitCode: make(chan int, 1),
				ctxErr:   make(chan error, 1),
				stdout:   newLockableBuffer(),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("Run() failed to run test command %v", err)
									}
									return 1 // no-zero exit code
								},
							},
						},
					},
				},
			},
			true,
			true,
		},
		{
			"missing command",
			fields{
				path:  "test",
				fExec: &FuncExec{},
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				stdout:   tt.fields.stdout,
				ctx:      context.Background(),
				ctxErr:   tt.fields.ctxErr,
				exitCode: tt.fields.exitCode,
				fExec:    tt.fields.fExec,
			}
			if tt.fields.ctxTimeout != 0 {
				ctx, cancel := context.WithTimeout(c.ctx, tt.fields.ctxTimeout)
				defer cancel()

				c.ctx = ctx
			}

			if err := c.Run(); (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantStart {
				return
			}

			// check that the function was started
			// make sure to wait long enough for the cmd func to finish
			time.Sleep(time.Millisecond * 10)
			gotStdout := c.stdout.(*lockableBuffer).String()
			wantStdout := "test was run"
			if gotStdout != wantStdout {
				t.Errorf("FuncCmd.Run() cmd func was not run, got stdout %s but wanted %s", gotStdout, wantStdout)
			}
		})
	}
}

func TestFuncCmd_CombinedOutput(t *testing.T) {
	type fields struct {
		path     string
		stdout   io.Writer
		stderr   io.Writer
		ctxErr   chan error
		exitCode chan int
		fExec    *FuncExec
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			"with stdout",
			fields{
				path:     "test",
				exitCode: make(chan int, 1),
				ctxErr:   make(chan error, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("CombinedOutput() failed to run test command %v", err)
									}
									return 0
								},
							},
						},
					},
				},
			},
			[]byte("test was run"),
			false,
		},
		{
			"with stderr",
			fields{
				path:     "test",
				exitCode: make(chan int, 1),
				ctxErr:   make(chan error, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stderr.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("CombinedOutput() failed to run test command %v", err)
									}
									return 0
								},
							},
						},
					},
				},
			},
			[]byte("test was run"),
			false,
		},
		{
			"both",
			fields{
				path:     "test",
				exitCode: make(chan int, 1),
				ctxErr:   make(chan error, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("CombinedOutput() failed to run test command %v", err)
									}
									_, err = fc.stderr.Write([]byte("there was an err"))
									if err != nil {
										t.Fatalf("CombinedOutput() failed to run test command %v", err)
									}
									return 0
								},
							},
						},
					},
				},
			},
			[]byte("test was runthere was an err"),
			false,
		},
		{
			"missing command",
			fields{
				path:     "test",
				exitCode: make(chan int, 1),
				ctxErr:   make(chan error, 1),
				fExec:    &FuncExec{},
			},
			[]byte{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				stdout:   tt.fields.stdout,
				stderr:   tt.fields.stderr,
				ctxErr:   tt.fields.ctxErr,
				exitCode: tt.fields.exitCode,
				fExec:    tt.fields.fExec,
			}
			got, err := c.CombinedOutput()
			if (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.CombinedOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncCmd.CombinedOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuncCmd_Environ(t *testing.T) {
	type fields struct {
		env   map[string]string
		fExec *FuncExec
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			"exec environ",
			fields{
				fExec: &FuncExec{
					envs: map[string]string{
						"TEST":  "true",
						"DEBUG": "false",
						"EXEC":  "10",
					},
				},
			},
			[]string{"DEBUG=false", "EXEC=10", "TEST=true"},
		},
		{
			"cmd environ",
			fields{
				env: map[string]string{
					"TEST":  "false",
					"DEBUG": "true",
					"EXEC":  "15",
				},
				fExec: &FuncExec{
					envs: map[string]string{
						"TEST":  "true",
						"DEBUG": "false",
						"EXEC":  "10",
					},
				},
			},
			[]string{"DEBUG=true", "EXEC=15", "TEST=false"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				env:   tt.fields.env,
				fExec: tt.fields.fExec,
			}
			if got := c.Environ(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncCmd.Environ() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuncCmd_Output(t *testing.T) {
	type fields struct {
		path     string
		ctxErr   chan error
		exitCode chan int
		fExec    *FuncExec
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			"with output",
			fields{
				path:     "test",
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("Output() failed to run test command %v", err)
									}
									return 0
								},
							},
						},
					},
				},
			},
			[]byte("test was run"),
			false,
		},
		{
			"with std error",
			fields{
				path:     "test",
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stderr.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("Output() failed to run test command %v", err)
									}
									return 1
								},
							},
						},
					},
				},
			},
			[]byte{},
			true,
		},
		{
			"missing cmd",
			fields{
				path:  "test",
				fExec: &FuncExec{},
			},
			[]byte{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				ctxErr:   tt.fields.ctxErr,
				exitCode: tt.fields.exitCode,
				fExec:    tt.fields.fExec,
			}
			got, err := c.Output()
			if (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.Output() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncCmd.Output() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestFuncCmd_StderrPipe(t *testing.T) {
	type fields struct {
		path     string
		stderr   io.Writer
		process  *os.Process
		ctxErr   chan error
		exitCode chan int
		fExec    *FuncExec
	}
	tests := []struct {
		name       string
		fields     fields
		wantOutput []byte
		wantErr    bool
	}{
		{
			"stderr already set",
			fields{
				stderr: &bytes.Buffer{},
			},
			nil,
			true,
		},
		{
			"process already started",
			fields{
				process: &os.Process{},
			},
			nil,
			true,
		},
		{
			"write to stderr",
			fields{
				path:     "test",
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stderr.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("StderrPipe() failed to run test command %v", err)
									}
									return 0
								},
							},
						},
					},
				},
			},
			[]byte("test was run"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				stderr:   tt.fields.stderr,
				process:  tt.fields.process,
				ctxErr:   tt.fields.ctxErr,
				exitCode: tt.fields.exitCode,
				fExec:    tt.fields.fExec,
			}
			got, err := c.StderrPipe()
			if (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.StderrPipe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantOutput == nil {
				return
			}

			err = c.Run()
			if err != nil {
				t.Errorf("FuncCmd.StderrPipe() filed to run test command %s", err)
				return
			}

			gotOutput, err := io.ReadAll(got)
			if err != nil {
				t.Errorf("FuncCmd.StderrPipe() failed to read off of buffer %s", err)
			}

			if !reflect.DeepEqual(gotOutput, tt.wantOutput) {
				t.Errorf("FuncCmd.StderrPipe() = %v, want %v", string(gotOutput), string(tt.wantOutput))
			}
		})
	}
}

func TestFuncCmd_StdoutPipe(t *testing.T) {
	type fields struct {
		path     string
		stdout   io.Writer
		process  *os.Process
		ctxErr   chan error
		exitCode chan int
		fExec    *FuncExec
	}
	tests := []struct {
		name       string
		fields     fields
		wantOutput []byte
		wantErr    bool
	}{
		{
			"stdout already set",
			fields{
				stdout: &bytes.Buffer{},
			},
			nil,
			true,
		},
		{
			"process already started",
			fields{
				process: &os.Process{},
			},
			nil,
			true,
		},
		{
			"write to stdout",
			fields{
				path:     "test",
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									_, err := fc.stdout.Write([]byte("test was run"))
									if err != nil {
										t.Fatalf("StdoutPipe() failed to run test command %v", err)
									}
									return 0
								},
							},
						},
					},
				},
			},
			[]byte("test was run"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				stdout:   tt.fields.stdout,
				process:  tt.fields.process,
				ctxErr:   tt.fields.ctxErr,
				exitCode: tt.fields.exitCode,
				fExec:    tt.fields.fExec,
			}
			got, err := c.StdoutPipe()
			if (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.StdoutPipe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantOutput == nil {
				return
			}

			err = c.Run()
			if err != nil {
				t.Errorf("FuncCmd.StdoutPipe() filed to run test command %s", err)
				return
			}

			gotOutput, err := io.ReadAll(got)
			if err != nil {
				t.Errorf("FuncCmd.StdoutPipe() failed to read off of buffer %s", err)
			}

			if !reflect.DeepEqual(gotOutput, tt.wantOutput) {
				t.Errorf("FuncCmd.StdoutPipe() = %v, want %v", string(gotOutput), string(tt.wantOutput))
			}
		})
	}
}

func TestFuncCmd_StdinPipe(t *testing.T) {
	type fields struct {
		path     string
		stdin    io.Reader
		process  *os.Process
		ctxErr   chan error
		exitCode chan int
		fExec    *FuncExec
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"stdin was already set",
			fields{
				stdin: &bytes.Buffer{},
			},
			true,
		},
		{
			"process already started",
			fields{
				process: &os.Process{},
			},
			true,
		},
		{
			"write to stdin",
			fields{
				path:     "test",
				ctxErr:   make(chan error, 1),
				exitCode: make(chan int, 1),
				fExec: &FuncExec{
					mux: &Mux{
						matchers: []muxMatcher{
							{
								pat: pat("test"),
								handler: func(fc *FuncCmd) int {
									gotInput, err := io.ReadAll(fc.stdin)
									if err != nil {
										return 1
									}
									if string(gotInput) != "test input" {
										return 1
									}
									return 0
								},
							},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				path:     tt.fields.path,
				stdin:    tt.fields.stdin,
				process:  tt.fields.process,
				ctxErr:   tt.fields.ctxErr,
				exitCode: tt.fields.exitCode,
				fExec:    tt.fields.fExec,
			}
			got, err := c.StdinPipe()
			if (err != nil) != tt.wantErr {
				t.Errorf("FuncCmd.StdinPipe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			_, err = got.Write([]byte("test input"))
			if err != nil {
				t.Errorf("FuncCmd.StdinPipe() could not write to stdin pipe %s", err)
				return
			}

			err = c.Run()
			if err != nil {
				t.Errorf("FuncCmd.StdinPipe() test cmd failed")
			}
		})
	}
}

func TestFuncCmd_SetEnv(t *testing.T) {
	type fields struct {
		env map[string]string
	}
	type args struct {
		env []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			"empty env vars",
			fields{
				env: make(map[string]string),
			},
			args{},
			nil,
		},
		{
			"nil env map",
			fields{},
			args{
				env: []string{"TEST_A=1", "TEST_B=two"},
			},
			[]string{"TEST_A=1", "TEST_B=two"},
		},
		{
			"golden path",
			fields{
				env: make(map[string]string),
			},
			args{
				env: []string{"TEST_A=1", "TEST_B=two", "EMPTY=", "DOUBLE=one=1"},
			},
			[]string{"DOUBLE=one=1", "EMPTY=", "TEST_A=1", "TEST_B=two"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FuncCmd{
				env: tt.fields.env,
			}
			c.SetEnv(tt.args.env)

			if env := c.Env(); !reflect.DeepEqual(env, tt.want) {
				t.Errorf("SetEnv() = %v, want %v", env, tt.want)
			}
		})
	}
}
