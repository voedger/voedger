/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CommandController(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name       string
		sp         CommandSP
		expectedPV *CommandPV
	}{
		{
			name: `stdout,exitcode=0`,
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
		},
		{
			name: `stderr,exitcode=0`,
			sp: CommandSP{
				Cmd:  `sh`,
				Args: []string{`-c`, `echo hello >&2`},
			},
			expectedPV: &CommandPV{
				Cmd:      `sh`,
				Args:     []string{`-c`, `echo hello >&2`},
				Stdout:   "",
				Stderr:   "hello",
				ExitCode: 0,
			},
		},
		{
			name: `stderr,exitcode=1`,
			sp: CommandSP{
				Cmd:  "pwd",
				Args: []string{`unused param`},
			},
			expectedPV: &CommandPV{
				Cmd:      "pwd",
				Args:     []string{`unused param`},
				Stdout:   "",
				Stderr:   "usage: pwd [-L | -P]",
				ExitCode: 1,
			},
		},
		{
			name: `exitcode=1`,
			sp: CommandSP{
				Cmd:  "ls",
				Args: []string{`/non/existent/directory`},
			},
			expectedPV: &CommandPV{
				Cmd:      "ls",
				Args:     []string{`/non/existent/directory`},
				Stdout:   "",
				Stderr:   "ls: /non/existent/directory: No such file or directory",
				ExitCode: 1,
			},
		},
		{
			name: `exitcode=0`,
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, pv, _ := CommandController(``, test.sp, struct{}{})

			require.Equal(t, *test.expectedPV, *pv)
		})
	}
}
