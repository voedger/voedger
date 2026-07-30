/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.  and Contributors
 * @author Maxim Geraskin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

/*
Links

- https://stackoverflow.com/questions/25190971/golang-copy-exec-output-to-log

*/

package exec

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/voedger/voedger/pkg/goutils/logger"
)

// https://github.com/b4b4r07/go-pipe/blob/master/README.md

// PipedExec allows to execute commands in pipe
type PipedExec struct {
	cmds []*pipedCmd
}

// Stderr redirection
const (
	StderrRedirectNone = iota
	StderrRedirectStdout
	StderrRedirectNull
)

type pipedCmd struct {
	stderrRedirection int
	cmd               *exec.Cmd
}

func (pe *PipedExec) command(name string, stderrRedirection int, args ...string) *PipedExec {
	//nolint:noctx
	cmd := exec.Command(name, args...)
	lastIdx := len(pe.cmds) - 1
	if lastIdx > -1 {
		var err error
		cmd.Stdin, err = pe.cmds[lastIdx].cmd.StdoutPipe()
		// notest
		if err != nil {
			panic(err)
		}
	} else {
		cmd.Stdin = os.Stdin
	}
	pe.cmds = append(pe.cmds, &pipedCmd{stderrRedirection, cmd})
	return pe
}

// GetCmd returns cmd with given index
func (pe *PipedExec) GetCmd(idx int) *exec.Cmd {
	return pe.cmds[idx].cmd
}

// Command adds a command to a pipe
func (pe *PipedExec) Command(name string, args ...string) *PipedExec {
	return pe.command(name, StderrRedirectNone, args...)
}

// WorkingDir sets working directory for the last command
func (pe *PipedExec) WorkingDir(wd string) *PipedExec {
	pipedCmd := pe.cmds[len(pe.cmds)-1]
	pipedCmd.cmd.Dir = wd
	return pe
}

// Wait until all cmds finish
func (pe *PipedExec) Wait() error {
	var firstErr error
	for _, cmd := range pe.cmds {
		err := cmd.cmd.Wait()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Start all cmds
func (pe *PipedExec) Start(out io.Writer, err io.Writer) error {
	for _, cmd := range pe.cmds {
		if cmd.stderrRedirection == StderrRedirectNone && err != nil {
			cmd.cmd.Stderr = err
		}
	}
	lastIdx := len(pe.cmds) - 1
	if lastIdx < 0 {
		return errors.New("empty command list")
	}
	if out != nil {
		pe.cmds[lastIdx].cmd.Stdout = out
	}

	for _, cmd := range pe.cmds {
		logger.Verbose(cmd.cmd.Path, cmd.cmd.Args)
		err := cmd.cmd.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

// Run starts the pipe
func (pe *PipedExec) Run(out io.Writer, err io.Writer) error {
	e := pe.Start(out, err)
	if e != nil {
		return e
	}
	return pe.Wait()
}

// RunToStrings runs the pipe and saves outputs to strings
func (pe *PipedExec) RunToStrings() (stdout string, stderr string, err error) {
	// _, stdoutw := io.Pipe()
	// _, stderrw := io.Pipe()

	var wg sync.WaitGroup

	lastCmd := pe.cmds[len(pe.cmds)-1]
	stdoutPipe, err := lastCmd.cmd.StdoutPipe()
	// notest
	if err != nil {
		return "", "", err
	}
	stderrPipe, err := lastCmd.cmd.StderrPipe()
	// notest
	if err != nil {
		return "", "", err
	}

	err = pe.Start(nil, nil)
	if err != nil {
		return "", "", err
	}

	wg.Go(func() {
		buf := new(bytes.Buffer)
		_, errr := buf.ReadFrom(stdoutPipe)
		// notestdept
		if errr != nil {
			panic(errr) // nolint
		}
		stdout = buf.String()
	})

	wg.Go(func() {
		buf := new(bytes.Buffer)
		_, errr := buf.ReadFrom(stderrPipe)
		// notestdept
		if errr != nil {
			panic(errr) // nolint
		}
		stderr = buf.String()
	})

	wg.Wait()

	return stdout, stderr, pe.Wait()
}
