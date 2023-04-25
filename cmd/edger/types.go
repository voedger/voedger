package main

import "os/exec"

type CommandSP struct {
	Cmd  string
	Args []string
}

func (c CommandSP) getCmd() *exec.Cmd {
	return exec.Command(c.Cmd, c.Args...)
}

type CommandPV struct {
	Cmd      string
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
}

type CommandState struct{}
