package pat

import (
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
