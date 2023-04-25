package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/logger"
)

func Test_CommandController(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	logger.Verbose("current working dir: ", wd)

	tests := []struct {
		name              string
		sp                CommandSP
		expectedPV        *CommandPV
		expectedStartTime *time.Time
	}{
		{
			name: `echo hello world`,
			sp: CommandSP{
				Cmd:  "echo",
				Args: []string{"hello", "world"},
			},
			expectedPV: &CommandPV{
				Cmd:      "echo",
				Args:     []string{"hello", "world"},
				Stdout:   "hello world",
				Stderr:   "",
				ExitCode: 0,
			},
			expectedStartTime: nil,
		},
		{
			name: `pwd`,
			sp: CommandSP{
				Cmd:  "pwd",
				Args: []string{},
			},
			expectedPV: &CommandPV{
				Cmd:      "pwd",
				Args:     []string{},
				Stdout:   wd,
				Stderr:   "",
				ExitCode: 0,
			},
			expectedStartTime: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, pv, nextStartTime := CommandController(``, test.sp, struct{}{})

			require.Equal(t, *test.expectedPV, *pv)
			require.Equal(t, test.expectedStartTime, nextStartTime)
		})
	}
}
