package cmd

import (
	"errors"
	"testing"
)

func TestCmdValidate(t *testing.T) {
	tests := []struct {
		name string
		// validation arguments
		args []string
		// want result
		wantName string
		wantErr  error
	}{
		{
			name:    "No arguments",
			wantErr: errors.New("invalid number of arguments: test <name> is a required argument"),
		},
		{
			name:    "2 arguments",
			args:    []string{"foo", "bar"},
			wantErr: errors.New("invalid number of arguments: test <name> is a required argument"),
		},
		{
			name:     "Valid argument",
			args:     []string{"mydeploy"},
			wantName: "mydeploy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := CmdOptions{Resource: "test"}
			err := opt.Validate(tt.args)
			if err != nil {
				if tt.wantErr == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("cmd.Validate(%v) wants %v, but got %v", tt.args, tt.wantErr, err)
				}
			} else {
				if tt.wantErr != nil {
					t.Fatalf("cmd.Validate(%v) wants %v, but no error", tt.args, tt.wantErr)
				}
			}
			if got, want := opt.Name, tt.wantName; got != want {
				t.Fatalf("opt.Name wants %v, but got %v", want, got)
			}
		})
	}
}
