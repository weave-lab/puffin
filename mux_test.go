package puffin

import (
	"testing"
)

func TestFuncMux_findHandler(t *testing.T) {
	type fields struct {
		matchers []muxMatcher
	}
	type args struct {
		cmd *FuncCmd
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantFunc bool
	}{
		{
			"no handlers",
			fields{},
			args{
				cmd: &FuncCmd{path: "test"},
			},
			false,
		},
		{
			"get handler",
			fields{
				matchers: []muxMatcher{
					{
						pat:     pat("test"),
						handler: func(cmd *FuncCmd) int { return 0 },
					},
					{
						pat:     pat("another"),
						handler: func(cmd *FuncCmd) int { return 0 },
					},
				},
			},
			args{
				cmd: &FuncCmd{path: "test"},
			},
			true,
		},
		{
			"full path",
			fields{
				matchers: []muxMatcher{
					{
						pat:     pat("/full/path/to/test"),
						handler: func(cmd *FuncCmd) int { return 0 },
					},
				},
			},
			args{
				cmd: &FuncCmd{path: "test"},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := &Mux{
				matchers: tt.fields.matchers,
			}
			got := mux.findHandler(tt.args.cmd)
			if (got != nil) != tt.wantFunc {
				t.Errorf("findFunc() got = %v, wantFunc %v", got, tt.wantFunc)
			}
		})
	}
}
