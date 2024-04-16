/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.  and Contributors
 * @author Maxim Geraskin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package exec_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/voedger/voedger/pkg/goutils/exec"
)

// Run a command and capture its output to strings
func ExamplePipedExec_RunToStrings() {

	stdout, stderr, err := new(exec.PipedExec).
		Command("echo", "RunToStrings").
		RunToStrings()

	if err != nil {
		fmt.Println("exec.PipedExec failed:", err, stderr)
	}

	fmt.Println(strings.TrimSpace(stdout))
	// Output: RunToStrings
}

// printf "1\n2\n3" | grep 2
func ExamplePipedExec_RunToStrings_pipe() {

	stdout, stderr, err := new(exec.PipedExec).
		Command("printf", `1\n2\n3`).
		Command("grep", "2").
		RunToStrings()

	if err != nil {
		fmt.Println("exec.PipedExec failed:", err, stderr)
	}

	fmt.Println(strings.TrimSpace(stdout))
	// Output: 2
}

// Run a command and use os.Stdout, os.Stderr
func ExamplePipedExec_Run() {

	err := new(exec.PipedExec).
		Command("echo", "Run").
		Run(os.Stdout, os.Stderr)

	if err != nil {
		fmt.Println("exec.PipedExec failed:", err)
	}

	// Output: Run
}
