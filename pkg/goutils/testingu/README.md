# testingu
`testingu` is a Go package designed to facilitate testing of command-line interface (CLI) applications. It provides tools to capture standard output and standard error, and to validate expected outcomes for different argument combinations.

## Installation
To install the `testingu` package, run:

```sh
go get -u github.com/voedger/voedger/pkg/goutils/testingu
```

## Usage
Below are the steps to use the testingu package to test your CLI applications.

### 1. Define Your Test Cases
Create a slice of CmdTestCase structs, each representing a test case. Each test case includes: 
- Name: The name of the test case.
- Args: The arguments to pass to the CLI tool.
- ExpectedErr: The expected error (if any).
- ExpectedErrPatterns: A list of expected substrings to find in the error message.
- ExpectedStdoutPatterns: A list of the expected substrings to find in stdout.
- ExpectedStderrPatterns: A list of the expected substrings to find in stderr.

```go
testCases := []testingu.CmdTestCase{
    {
        Name:               "wrong number of arguments",
        Args:               []string{"cli", "init", "--dry-run", "arg1", "arg2", "arg3"},
        ExpectedErrPattern: "1 arg(s)",
    },
    {
        Name:               "unknown flag",
        Args:               []string{"cli", "init", "--dry-run", "--unknown_flag", "arg1"},
        ExpectedErrPattern: "unknown flag",
    },
    {
        Name:           "help",
        Args:           []string{"cli", "help"},
        ExpectedStdout: "cli [command]",
    },
    {
        Name:               "unknown command",
        Args:               []string{"cli", "unknown_command"},
        ExpectedErrPattern: "",
        ExpectedStdout:     "cli [command]",
    },
}
```

### 2. Implement the CLI Command Execution Function
Create a function that executes your CLI command. This function should accept args and version as parameters and return an error.

```go
func execRootCmd(args []string, ver string) error {
    params := &vpmParams{}
    rootCmd := cobrau.PrepareRootCmd(
    "cli",
    "",
    args,
    ver,
    newInitCmd(params),
    newTidyCmd(params),
    newBuildCmd(params),
    )
    rootCmd.InitDefaultHelpCmd()
    rootCmd.InitDefaultCompletionCmd()
    return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
``` 
### 3. Run the Test Cases
Use the RunCmdTestCases function from the `testingu` package to run your test cases.

```go
func TestCommandMessaging(t *testing.T) {
	testCases := []testingu.CmdTestCase{
		{
			Name:               "wrong number of arguments",
			Args:               []string{"cli", "init", "--dry-run", "arg1", "arg2", "arg3"},
			ExpectedErrPattern: "1 arg(s)",
		},
		{
			Name:               "unknown flag",
			Args:               []string{"cli", "init", "--dry-run", "--unknown_flag", "arg1"},
			ExpectedErrPattern: "unknown flag",
		},
		{
			Name:           "help",
			Args:           []string{"cli", "help"},
			ExpectedStdout: "cli [command]",
		},
		{
			Name:               "unknown command",
			Args:               []string{"cli", "unknown_command"},
			ExpectedErrPattern: "",
			ExpectedStdout:     "cli [command]",
		},
	}

	testingu.RunCmdTestCases(t, execRootCmd, testCases)
}
```

### 4. Run Your Tests
Execute your tests with the Go testing tool:

```sh
go test
```

## Types

### CmdTestCase
A struct representing a test case for a CLI command.

#### Fields:
- Name string: The name of the test case.
- Args []string: The arguments to pass to the CLI command.
- ExpectedErr error: The expected error (if any).
- ExpectedErrPatterns []string: A list of expected substrings to find in the error message.
- ExpectedStdoutPatterns []string: A list of the expected substrings to find in stdout.
- ExpectedStderrPatterns []string: A list of the expected substrings to find in stderr.

## Functions

### RunCmdTestCases
Runs a series of command-line test cases.

#### Parameters:
- t *testing.T: The testing object.
- execute func(args []string, version string) error: The function to execute the CLI command.
- testCases []CmdTestCase: The test cases to run.
- version string: The version of the CLI command.

#### Returns:
No return value.

### CaptureStdoutStderr
Captures the standard output and standard error of a function.

#### Parameters:
f func() error: The function to capture output from.

#### Returns:
- stdout string: The captured standard output.
- stderr string: The captured standard error.
- err error: The error returned by the function.
