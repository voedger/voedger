package main

import (
	"bytes"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/untillpro/goutils/logger"
)

func CommandController(_ string, sp CommandSP, _ CommandState) (_ *CommandState, pv *CommandPV, _ *time.Time) {
	cmd := sp.getCmd()

	// Prepare a buffer to store the command output
	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	// Run the command and wait for its completion
	err := cmd.Run()

	var exitCode int
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// The command failed with a non-zero exit code
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			logger.Verbose("failed to execute command: %v", err)
		}
	} else {
		// The command succeeded with a zero exit code
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	stderr := strings.TrimSpace(stderrBuffer.String())
	stdout := strings.TrimSpace(stdoutBuffer.String())
	pv = &CommandPV{
		Cmd:      sp.Cmd,
		Args:     sp.Args,
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}
	return
}
