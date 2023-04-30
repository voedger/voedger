/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

import (
	"bytes"
	"os/exec"
	"strings"
	"syscall"
	"time"
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
