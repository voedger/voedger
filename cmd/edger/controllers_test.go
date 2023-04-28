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
		name             string
		sp               CommandSP
		expectedStdout   string
		expectedStderr   string
		expectedExitCode int
	}{
		{
			name: `stdout,exitcode=0`,
			sp: CommandSP{
				Cmd:  "echo",
				Args: []string{"hello", "world"},
			},
			expectedStdout:   "hello world",
			expectedStderr:   "",
			expectedExitCode: 0,
		},
		{
			name: `stderr,exitcode=0`,
			sp: CommandSP{
				Cmd:  `sh`,
				Args: []string{`-c`, `echo hello >&2`},
			},
			expectedStdout:   "",
			expectedStderr:   "hello",
			expectedExitCode: 0,
		},
		{
			name: `stderr,exitcode=1`,
			sp: CommandSP{
				Cmd:  "ls",
				Args: []string{`/non/existent/directory`},
			},
			expectedStdout:   "",
			expectedStderr:   "ls: /non/existent/directory: No such file or directory",
			expectedExitCode: 1,
		},
		{
			name: `exitcode=0`,
			sp: CommandSP{
				Cmd:  "pwd",
				Args: []string{},
			},
			expectedStdout:   wd,
			expectedStderr:   "",
			expectedExitCode: 0,
		},
		{
			name: `exitcode=1`,
			sp: CommandSP{
				Cmd:  "ls",
				Args: []string{`/non/existent/directory`, `>/dev/null 2>&1`},
			},
			expectedStdout:   "",
			expectedStderr:   "ls: /non/existent/directory: No such file or directory",
			expectedExitCode: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, pv, _ := CommandController(``, test.sp, struct{}{})

			require.Equal(t, len(test.expectedStdout) > 0, len(pv.Stdout) > 0)
			require.Equal(t, len(test.expectedStderr) > 0, len(pv.Stderr) > 0)
			require.Equal(t, test.expectedExitCode, pv.ExitCode)
		})
	}
}
