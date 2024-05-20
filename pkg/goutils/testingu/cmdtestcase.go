/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package testingu

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
)

type CmdTestCase struct {
	Name                   string
	Args                   []string
	Version                string
	ExpectedErr            error
	ExpectedErrPatterns    []string
	ExpectedStdoutPatterns []string
	ExpectedStderrPatterns []string
}

func RunCmdTestCases(t *testing.T, execute func(args []string, version string) error, testCases []CmdTestCase, version string) {
	// notestdept
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			f := func() error {
				return execute(tc.Args, version)
			}

			stdout, stderr, err := CaptureStdoutStderr(f)
			log.Println("stdout:", stdout)
			log.Println("stderr:", stderr)

			checkOutput(t, tc.ExpectedStdoutPatterns, stdout, "stdout")
			checkOutput(t, tc.ExpectedStderrPatterns, stderr, "stderr")

			checkError(t, tc.ExpectedErr, tc.ExpectedErrPatterns, err)
		})
	}
}

func checkError(t *testing.T, expectedErr error, expectedErrPatterns []string, actualErr error) {
	// notestdept
	t.Helper()
	if expectedErr != nil || len(expectedErrPatterns) > 0 {
		for _, expectedErrPattern := range expectedErrPatterns {
			if actualErr == nil {
				t.Errorf("error was not returned as expected")
				return
			}
			if expectedErr != nil && !errors.Is(actualErr, expectedErr) {
				t.Errorf("wrong error was returned: expected `%v`, got `%v`", expectedErr, actualErr)
				return
			}
			if len(expectedErrPattern) > 0 && !strings.Contains(actualErr.Error(), expectedErrPattern) {
				t.Errorf("wrong error was returned: expected pattern `%v`, got `%v`", expectedErrPattern, actualErr.Error())
				return
			}
		}
	} else if actualErr != nil {
		t.Errorf("unexpected error was returned: %v", actualErr)
	}
}

func checkOutput(t *testing.T, expectedPatterns []string, actual, outputTitle string) {
	t.Helper()
	for _, expectedPattern := range expectedPatterns {
		switch {
		case len(actual) == 0 && len(expectedPattern) > 0:
			t.Errorf("%s: expected pattern `%v`, actual is nothing", outputTitle, expectedPattern)
		case !strings.Contains(actual, expectedPattern) && len(expectedPattern) > 0:
			t.Errorf("%s: expected pattern `%v`, actual `%v`", outputTitle, expectedPattern, actual)
		case len(actual) > 0 && len(expectedPattern) == 0:
			t.Errorf("%s: expected nothing, got `%v`", outputTitle, actual)
		}
	}
}

// https://go.dev/play/p/Fzj1k7jul7z

func CaptureStdoutStderr(f func() error) (stdout string, stderr string, err error) {

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		// notestdept
		return
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		// notestdept
		return
	}

	{
		origStdout := os.Stdout
		os.Stdout = stdoutWriter
		defer func() { os.Stdout = origStdout }()
	}
	{
		origStderr := os.Stderr
		os.Stderr = stderrWriter
		defer func() { os.Stderr = origStderr }()
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		var b bytes.Buffer
		defer wg.Done()
		_, _ = io.Copy(&b, stdoutReader)
		stdout = b.String()
	}()
	wg.Add(1)
	go func() {
		var b bytes.Buffer
		defer wg.Done()
		_, _ = io.Copy(&b, stderrReader)
		stderr = b.String()
	}()

	err = f()
	stderrWriter.Close()
	stdoutWriter.Close()
	wg.Wait()
	return

}
