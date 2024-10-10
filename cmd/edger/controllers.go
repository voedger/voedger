/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	goutilsExec "github.com/voedger/voedger/pkg/goutils/exec"
)

// nolint
func CommandController(_ string, sp CommandSP, _ CommandState) (_ *CommandState, _ *CommandPV, _ *time.Time) {
	cmd := exec.Command(sp.Cmd, sp.Args...)

	// Prepare a buffer to store the command output
	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	// Run the command and wait for its completion
	err := cmd.Run()
	stderr := strings.TrimSpace(stderrBuffer.String())
	stdout := strings.TrimSpace(stdoutBuffer.String())

	var exitCode int
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// The command failed with a non-zero exit code
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			pv := &CommandPV{
				Cmd:      sp.Cmd,
				Args:     sp.Args,
				Stdout:   stdout,
				Stderr:   err.Error(),
				ExitCode: cmd.ProcessState.ExitCode(),
			}
			return nil, pv, nil
		}
	} else {
		// The command succeeded with a zero exit code
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	pv := &CommandPV{
		Cmd:      sp.Cmd,
		Args:     sp.Args,
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}
	return nil, pv, nil
}

// nolint
func DockerController(projectName string, sp DockerSP, _ DockerState) (*DockerState, *DockerPV, *time.Time) {
	attemptOfStart := time.Now()
	if err := composeUp(projectName, sp.ComposeText); err != nil {
		return nil, newDockerPV(err, sp.Version, attemptOfStart), nil
	}
	return nil, nil, nil
}

func composeUp(projectName, composeText string) error {
	return new(goutilsExec.PipedExec).
		Command("echo", composeText).
		Command("docker-compose", "-f", "-", "-p", projectName, "up", "-d", "--remove-orphans").
		Run(os.Stdout, os.Stderr)
}

func rmExitedContainers(projectName string) error {
	containerIDs, err := getContainerIDs(projectName, map[string]string{"status": "exited"}, "-a")
	if err != nil {
		return err
	}

	for _, containerID := range containerIDs {
		if err := exec.Command("docker", "rm", containerID).Run(); err != nil {
			return err
		}
	}
	return nil
}

// nolint
func cleanUp(projectName string) error {
	if err := stopContainers(projectName); err != nil {
		return err
	}
	return rmExitedContainers(projectName)
}

func buildDockerArgs(command, projectName string, filter map[string]string, args ...string) (allArgs []string) {
	allArgs = make([]string, 0)
	allArgs = append(allArgs, command)
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, "-f", "name="+projectName)
	for k, v := range filter {
		allArgs = append(allArgs, "-f", fmt.Sprintf("%s=%s", k, v))
	}
	return
}

// nolint
func getContainerIDs(projectName string, filter map[string]string, args ...string) ([]string, error) {
	args = append(args, "-q")
	cmd := exec.Command("docker", buildDockerArgs("ps", projectName, filter, args...)...)

	var stdoutBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	stdout := strings.TrimSpace(stdoutBuffer.String())

	if len(stdout) > 0 {
		return strings.Split(stdout, "\n"), nil
	}
	return nil, nil
}

func stopContainers(projectName string) error {
	containerIDs, err := getContainerIDs(projectName, nil)
	if err != nil {
		return err
	}

	for _, containerID := range containerIDs {
		if err := exec.Command("docker", "stop", containerID).Run(); err != nil {
			return err
		}
	}
	return nil
}

// nolint
func dockerContainers(projectName string) (state dockerContainerInfoList, err error) {
	cmd := exec.Command("docker", "ps", "-a", "-f", fmt.Sprintf("name=%s", projectName), "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	container := dockerContainerFullInfo{}
	state = make([]dockerContainerInfo, 0)
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		if err := json.Unmarshal([]byte(line), &container); err != nil {
			return nil, err
		}

		state = append(state, dockerContainerInfo{
			Name:  container.Names,
			Image: container.Image,
			IsUp:  strings.HasPrefix(container.Status, "Up"),
		})
	}

	return state, nil
}
