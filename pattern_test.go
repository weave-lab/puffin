package puffin

import "testing"

func Test_pat_Match(t *testing.T) {
	type args struct {
		cmd *FuncCmd
	}
	tests := []struct {
		name string
		p    pat
		args args
		want bool
	}{
		{
			"don't match",
			pat("fail"),
			args{
				cmd: &FuncCmd{
					path: "test",
				},
			},
			false,
		},
		{
			"match",
			pat("test"),
			args{
				cmd: &FuncCmd{
					path: "test",
				},
			},
			true,
		},
		{
			"wildcard match",
			pat("*"),
			args{
				cmd: &FuncCmd{
					path: "test",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Match(tt.args.cmd); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
