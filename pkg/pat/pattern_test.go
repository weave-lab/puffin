package pat

import (
	"reflect"
	"testing"

	"github.com/weave-lab/puffin"
)

func TestPattern_Match(t *testing.T) {
	type fields struct {
		Cmd  string
		Args []string
	}
	type args struct {
		cmd *puffin.FuncCmd
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			"match command name",
			fields{
				Cmd: "test",
			},
			args{
				cmd: puffinCmd("test"),
			},
			true,
		},
		{
			"wildcard match",
			fields{
				Cmd: "*",
			},
			args{
				cmd: puffinCmd("test"),
			},
			true,
		},
		{
			"match with arguments",
			fields{
				Cmd:  "test",
				Args: []string{"arg1", "arg2"},
			},
			args{
				cmd: puffinCmd("test", "arg2", "arg1"),
			},
			true,
		},
		{
			"don't match command name",
			fields{
				Cmd: "fail",
			},
			args{
				cmd: puffinCmd("test"),
			},
			false,
		},
		{
			"dont match with args",
			fields{
				Cmd:  "test",
				Args: []string{"arg1"},
			},
			args{
				cmd: puffinCmd("test", "dontMatch"),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pattern{
				Cmd:  tt.fields.Cmd,
				Args: tt.fields.Args,
			}
			if got := p.Match(tt.args.cmd); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func puffinCmd(cmd string, args ...string) *puffin.FuncCmd {
	return puffin.NewFuncExec(nil).Command(cmd, args...).(*puffin.FuncCmd)
}

func TestNew(t *testing.T) {
	type args struct {
		cmd  string
		args []string
	}
	tests := []struct {
		name      string
		args      args
		want      *Pattern
		match     []*puffin.FuncCmd
		dontMatch []*puffin.FuncCmd
	}{
		{
			"no args",
			args{
				cmd: "test",
			},
			&Pattern{
				Cmd: "test",
			},
			[]*puffin.FuncCmd{
				puffinCmd("test"),
				puffinCmd("test", "arg1", "arg2"),
			},
			[]*puffin.FuncCmd{
				puffinCmd("go", "arg1", "arg2"),
				puffinCmd("cmd"),
			},
		},
		{
			"with args",
			args{
				cmd:  "test",
				args: []string{"arg2", "arg1"},
			},
			&Pattern{
				Cmd:  "test",
				Args: []string{"arg2", "arg1"},
			},
			[]*puffin.FuncCmd{
				puffinCmd("test", "arg1", "arg2"),
				puffinCmd("test", "arg1", "arg2", "arg3"),
			},
			[]*puffin.FuncCmd{
				puffinCmd("test"),
				puffinCmd("test", "arg0"),
			},
		},
		{
			"args only",
			args{
				cmd:  "*",
				args: []string{"arg2", "arg1"},
			},
			&Pattern{
				Cmd:  "*",
				Args: []string{"arg2", "arg1"},
			},
			[]*puffin.FuncCmd{
				puffinCmd("test", "arg2", "arg1", "arg0"),
				puffinCmd("git", "arg2", "arg1"),
				puffinCmd("go", "arg1", "arg2"),
			},
			[]*puffin.FuncCmd{
				puffinCmd("fail"),
				puffinCmd("go", "build", "."),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.cmd, tt.args.args...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}

			for _, match := range tt.match {
				if !got.Match(match) {
					t.Errorf("New() failed to match %v", match)
				}
			}

			for _, match := range tt.dontMatch {
				if got.Match(match) {
					t.Errorf("New() invalid match %v", match)
				}
			}
		})
	}
}
