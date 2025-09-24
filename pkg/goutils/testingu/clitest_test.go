/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package testingu

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCLITests(t *testing.T) {
	testError := errors.New("comprehensive error")
	tests := []struct {
		name      string
		execute   func([]string, string) error
		testCases []CmdTestCase
	}{
		{
			name: "stdout pattern matches output",
			execute: func(args []string, version string) error {
				fmt.Print("this contains expected text")
				return nil
			},
			testCases: []CmdTestCase{
				{
					Name:                   "stdout pattern match",
					Args:                   []string{},
					ExpectedStdoutPatterns: []string{"expected"},
				},
			},
		},
		{
			name: "multiline output",
			execute: func(args []string, version string) error {
				fmt.Println("line 1")
				fmt.Println("line 2")
				fmt.Fprintln(os.Stderr, "error line 1")
				fmt.Fprintln(os.Stderr, "error line 2")
				return nil
			},
			testCases: []CmdTestCase{
				{
					Name:                   "multiline",
					Args:                   []string{},
					ExpectedStdoutPatterns: []string{"line 1", "line 2"},
					ExpectedStderrPatterns: []string{"error line 1", "error line 2"},
				},
			},
		},
		{
			name: "stdout,stderr and error",
			execute: func(args []string, version string) error {
				fmt.Print("stdout output")
				fmt.Fprint(os.Stderr, "stderr output")
				return testError
			},
			testCases: []CmdTestCase{
				{
					Name:                   "comprehensive test",
					Args:                   []string{"comprehensive"},
					ExpectedStdoutPatterns: []string{"stdout output"},
					ExpectedStderrPatterns: []string{"stderr output"},
					ExpectedErrPatterns:    []string{"comprehensive error"},
					ExpectedErr:            testError,
				},
			},
		},
		{
			name: "successful execution with stderr patterns",
			execute: func(args []string, version string) error {
				fmt.Fprint(os.Stderr, "error message")
				return nil
			},
			testCases: []CmdTestCase{
				{
					Name:                   "stderr test",
					Args:                   []string{},
					ExpectedStderrPatterns: []string{"error message"},
				},
			},
		},
		{
			name: "empty error pattern should be ignored",
			execute: func(args []string, version string) error {
				return errors.New("some error")
			},
			testCases: []CmdTestCase{
				{
					Name:                "empty pattern case",
					Args:                []string{},
					ExpectedErrPatterns: []string{""},
				},
			},
		},
		{
			name: "error pattern expected and matching error returned",
			execute: func(args []string, version string) error {
				return errors.New("this contains pattern")
			},
			testCases: []CmdTestCase{
				{
					Name:                "pattern match case",
					Args:                []string{},
					ExpectedErrPatterns: []string{"contains pattern"},
				},
			},
		},
		{
			name: "multiple error patterns with matching error",
			execute: func(args []string, version string) error {
				return errors.New("error with pattern1 and pattern2")
			},
			testCases: []CmdTestCase{
				{
					Name:                "multiple patterns case",
					Args:                []string{},
					ExpectedErrPatterns: []string{"pattern1", "pattern2"},
				},
			},
		},
		{
			name: "no error expected and no error returned",
			execute: func(args []string, version string) error {
				return nil
			},
			testCases: []CmdTestCase{
				{
					Name: "success case",
					Args: []string{},
				},
			},
		},
		{
			name: "multiple test cases",
			execute: func(args []string, version string) error {
				if len(args) > 0 && args[0] == "fail" {
					return errors.New("failure")
				}
				if len(args) > 0 && args[0] == "success" {
					fmt.Print("output 1")
				}
				return nil
			},
			testCases: []CmdTestCase{
				{
					Name:                   "success case",
					Args:                   []string{"success"},
					ExpectedStdoutPatterns: []string{"output 1"},
				},
				{
					Name:                "failure case",
					Args:                []string{"fail"},
					ExpectedErrPatterns: []string{"failure"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RunCLITests(t, tt.execute, tt.testCases, "")
		})
	}
}

func TestRunCLITestsWithVersion(t *testing.T) {
	tests := []struct {
		name      string
		execute   func([]string, string) error
		testCases []CmdTestCase
		version   string
	}{
		{
			name: "version parameter is passed correctly",
			execute: func(args []string, version string) error {
				require := require.New(t)
				require.Equal("v2.1.0", version)
				if len(args) > 0 && args[0] == "version" {
					fmt.Printf("version %s", version)
				}
				return nil
			},
			testCases: []CmdTestCase{
				{
					Name:                   "version command",
					Args:                   []string{"version"},
					ExpectedStdoutPatterns: []string{"version v2.1.0"},
				},
			},
			version: "v2.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RunCLITests(t, tt.execute, tt.testCases, tt.version)
		})
	}
}

func TestCmdTestCase(t *testing.T) {
	tests := []struct {
		name     string
		testCase CmdTestCase
		validate func(*testing.T, CmdTestCase)
	}{
		{
			name: "complete test case structure",
			testCase: CmdTestCase{
				Name:                   "complete test",
				Args:                   []string{"arg1", "arg2"},
				Version:                "v1.0.0",
				ExpectedErr:            errors.New("expected error"),
				ExpectedErrPatterns:    []string{"pattern1", "pattern2"},
				ExpectedStdoutPatterns: []string{"stdout1", "stdout2"},
				ExpectedStderrPatterns: []string{"stderr1", "stderr2"},
			},
			validate: func(t *testing.T, tc CmdTestCase) {
				require := require.New(t)
				require.Equal("complete test", tc.Name)
				require.Equal([]string{"arg1", "arg2"}, tc.Args)
				require.Equal("v1.0.0", tc.Version)
				require.Len(tc.ExpectedErrPatterns, 2)
				require.Len(tc.ExpectedStdoutPatterns, 2)
				require.Len(tc.ExpectedStderrPatterns, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.testCase)
		})
	}
}

func TestRunCLITestsFailureScenarios(t *testing.T) {
	testError := errors.New("comprehensive error")
	tests := []struct {
		name     string
		execute  func([]string, string) error
		testCase CmdTestCase
	}{
		{
			name: "stdout pattern does not match",
			execute: func(args []string, version string) error {
				fmt.Print("actual output")
				return nil
			},
			testCase: CmdTestCase{
				Name:                   "pattern mismatch",
				Args:                   []string{},
				ExpectedStdoutPatterns: []string{"expected different output"},
			},
		},
		{
			name: "stderr pattern does not match",
			execute: func(args []string, version string) error {
				fmt.Fprint(os.Stderr, "actual error")
				return nil
			},
			testCase: CmdTestCase{
				Name:                   "stderr pattern mismatch",
				Args:                   []string{},
				ExpectedStderrPatterns: []string{"expected different error"},
			},
		},
		{
			name: "expected error but no error returned",
			execute: func(args []string, version string) error {
				return nil // No error returned
			},
			testCase: CmdTestCase{
				Name:                "missing expected error",
				Args:                []string{},
				ExpectedErrPatterns: []string{"expected error"},
			},
		},
		{
			name: "error pattern does not match",
			execute: func(args []string, version string) error {
				return errors.New("actual error message")
			},
			testCase: CmdTestCase{
				Name:                "error pattern mismatch",
				Args:                []string{},
				ExpectedErrPatterns: []string{"expected different pattern"},
			},
		},
		{
			name: "unexpected error when none expected",
			execute: func(args []string, version string) error {
				return errors.New("unexpected error")
			},
			testCase: CmdTestCase{
				Name: "unexpected error",
				Args: []string{},
				// No expected error or patterns
			},
		},
		{
			name: "expected empty stdout but got output",
			execute: func(args []string, version string) error {
				fmt.Print("unexpected output")
				return nil
			},
			testCase: CmdTestCase{
				Name:                   "unexpected stdout",
				Args:                   []string{},
				ExpectedStdoutPatterns: []string{""},
				ExpectedErr:            nil,
			},
		},
		{
			name: "expected empty stderr but got output",
			execute: func(args []string, version string) error {
				fmt.Fprint(os.Stderr, "unexpected error output")
				return nil
			},
			testCase: CmdTestCase{
				Name:                   "unexpected stderr",
				Args:                   []string{},
				ExpectedStderrPatterns: []string{""},
			},
		},
		{
			name: "expected another error",
			execute: func(args []string, version string) error {
				return testError
			},
			testCase: CmdTestCase{
				Name:                "unexpected stderr",
				Args:                []string{},
				ExpectedErrPatterns: []string{"another error pattern"},
				ExpectedErr:         errors.New("another error pattern"),
			},
		},
		{
			name: "expected output but actual nothing",
			execute: func(args []string, version string) error {
				return nil
			},
			testCase: CmdTestCase{
				Name:                   "unexpected stderr",
				Args:                   []string{},
				ExpectedStdoutPatterns: []string{"expected output"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := &mockTestingT{TB: &testing.T{}}
			RunCLITests(mockT, tt.execute, []CmdTestCase{tt.testCase}, "")
			require.True(t, mockT.Failed())
		})
	}
}

// mockTestingT embeds testing.TB and overrides failure methods to track failures
type mockTestingT struct {
	testing.TB
}

func (m *mockTestingT) Run(name string, f func(t *testing.T)) bool {
	f(m.TB.(*testing.T))
	return true
}
