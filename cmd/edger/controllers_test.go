/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
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
			require.Equal(t, test.expectedExitCode == 0, pv.ExitCode == 0)
		})
	}
}

func TestDockerController_AllContainersRunning(t *testing.T) {
	if testing.Short() {
		t.Skip(`skipping test in short mode`)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows.")
	}

	projectName := `my`
	sp := DockerSP{
		Version: "1.0",
		ComposeText: `
version: "3.7"
services:
    redis:
        image: 'redis:7.0.11-alpine'
        restart: always
    nginx:
        image: 'nginx:1.23.4'
        restart: always
`,
	}

	delimiter := dockerContainerNameDelimiter()
	expectedNewState := dockerContainerInfoList{
		{
			Name:  fmt.Sprintf("my%sredis%s1", delimiter, delimiter),
			Image: "redis:7.0.11-alpine",
			IsUp:  true,
		},
		{
			Name:  fmt.Sprintf("my%snginx%s1", delimiter, delimiter),
			Image: "nginx:1.23.4",
			IsUp:  true,
		},
	}

	err := cleanUp(projectName)
	require.NoError(t, err)
	DockerController(projectName, sp, struct{}{})

	newState, err := dockerContainers(projectName)
	require.NoError(t, err)

	sort.Sort(expectedNewState)
	sort.Sort(newState)
	require.Equal(t, len(expectedNewState), len(newState))
	require.Equal(t, expectedNewState, newState)

	err = cleanUp(projectName)
	require.NoError(t, err)
}

func TestDockerController_UpdateImage(t *testing.T) {
	if testing.Short() {
		t.Skip(`Skipping test in short mod`)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows.")
	}
	projectName := `my`
	sp := DockerSP{
		Version: "1.0",
		ComposeText: `
version: "3.7"
services:
    redis:
        image: 'redis:7.0.4-alpine'
        restart: always`,
	}

	delimiter := dockerContainerNameDelimiter()
	expectedNewState := dockerContainerInfoList{
		{
			Name:  fmt.Sprintf("my%sredis%s1", delimiter, delimiter),
			Image: "redis:7.0.4-alpine",
			IsUp:  true,
		},
	}

	err := cleanUp(projectName)
	require.NoError(t, err)

	_, pv, _ := DockerController(projectName, sp, DockerState{})
	if pv != nil {
		require.NoError(t, pv.Err)
	}

	newState, err := dockerContainers(projectName)
	require.NoError(t, err)

	require.Equal(t, len(expectedNewState), len(newState))
	//require.Equal(t, expectedNewState.states, newState.states)

	// updating image to version 7.0.11-alpine
	sp.ComposeText = `
version: "3.7"
services:
    redis:
        image: 'redis:7.0.11-alpine'
        restart: always`

	expectedNewState[0].Image = "redis:7.0.11-alpine"

	DockerController(projectName, sp, DockerState{})

	newState, err = dockerContainers(projectName)
	require.NoError(t, err)

	require.Equal(t, len(expectedNewState), len(newState))
	//require.Equal(t, expectedNewState.states, newState.states)

	// adding nginx service
	err = cleanUp(projectName)
	require.NoError(t, err)

	sp.ComposeText = `
version: "3.7"
services:
    redis:
        image: 'redis:7.0.11-alpine'
        restart: always
    nginx:
        image: 'nginx:1.23.4'
        restart: always`

	DockerController(projectName, sp, DockerState{})

	newState, err = dockerContainers(projectName)
	require.NoError(t, err)

	expectedNewState = dockerContainerInfoList{
		{
			Name:  fmt.Sprintf("my%sredis%s1", delimiter, delimiter),
			Image: "redis:7.0.11-alpine",
			IsUp:  true,
		},
		{
			Name:  fmt.Sprintf("my%snginx%s1", delimiter, delimiter),
			Image: "nginx:1.23.4",
			IsUp:  true,
		},
	}

	sort.Sort(expectedNewState)
	sort.Sort(newState)
	require.Equal(t, len(expectedNewState), len(newState))
	require.Equal(t, expectedNewState, newState)

	err = cleanUp(projectName)
	require.NoError(t, err)
}

func TestDockerController_InvalidComposeFile(t *testing.T) {
	if testing.Short() {
		t.Skip(`Skipping test in short mod`)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows.")
	}
	
	projectName := `my`
	sp := DockerSP{
		Version:     "1.0",
		ComposeText: "this is not valid YAML",
	}

	err := cleanUp(projectName)
	require.NoError(t, err)

	_, pv, _ := DockerController(projectName, sp, DockerState{})
	if pv != nil {
		require.Error(t, pv.Err)
	}

	err = cleanUp(projectName)
	require.NoError(t, err)
}

func dockerContainerNameDelimiter() string {
	switch runtime.GOOS {
	case `linux`:
		return `_`
	default:
		return `-`
	}
}
