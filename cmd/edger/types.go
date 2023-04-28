/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

// Command related types
type (
	CommandSP struct {
		Cmd  string
		Args []string
	}
	CommandPV struct {
		Cmd      string
		Args     []string
		Stdout   string
		Stderr   string
		ExitCode int
	}
	CommandState struct{}
)
