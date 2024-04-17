/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package testingu

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

type RootTestCase struct {
	Name               string
	Args               []string
	Version            string
	ExpectedErr        error
	ExpectedErrPattern string
}

func RunRootTestCases(t *testing.T, execute func(args []string, version string) error, testCases []RootTestCase) {
	// notestdept
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()
			f := func() error {
				return execute(tc.Args, tc.Version)
			}

			_, _, err := CaptureStdoutStderr(f)

			checkError(t, tc.ExpectedErr, tc.ExpectedErrPattern, err)
		})
	}
}

func checkError(t *testing.T, expectedErr error, expectedErrPattern string, actualErr error) {
	// notestdept
	t.Helper()
	if expectedErr != nil || len(expectedErrPattern) > 0 {
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
	} else if actualErr != nil {
		t.Errorf("unexpected error was returned: %v", actualErr)
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
