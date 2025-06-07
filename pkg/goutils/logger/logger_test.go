/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package logger_test

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
)

func Test_BasicUsage(t *testing.T) {

	// "Hello world"
	{
		logger.Error("Error", "arg1", "arg2")
		logger.Warning("My warning")
		logger.Info("My info")

		// IsVerbose() is used to avoid unnecessary calculations
		if logger.IsVerbose() {
			logger.Verbose("!!! You should NOT see this verbose message since default level is INFO")
		}

		// IsTrace() is used to avoid unnecessary calculations
		if logger.IsTrace() {
			logger.Trace("!!! You should NOT see this trace message since default level is INFO")
		}
	}

	// Changing LogLevel
	{
		logger.SetLogLevel(logger.LogLevelTrace)
		if logger.IsTrace() {
			logger.Trace("Now you should see my Trace")
		}
		if logger.IsVerbose() {
			logger.Verbose("Now you should see my Verbose")
		}
		logger.SetLogLevel(logger.LogLevelError)
		logger.Trace("!!! You should NOT see my Trace")
		logger.Warning("!!! You should NOT see my warning")
		logger.SetLogLevel(logger.LogLevelInfo)
		logger.Warning("You should see my warning")
		logger.Warning("You should see my info")
	}

	// Let see how it looks when using from methods
	{
		m := mystruct{}
		m.iWantToLog()
	}
}

func Test_BasicUsage_SetLogLevelWithRestore(t *testing.T) {

	// Log level is set to LogLevelTrace and then restored to the previous value
	defer logger.SetLogLevelWithRestore(logger.LogLevelTrace)()

	logger.Trace("You SHOULD see this trace")

}

func Test_SetLogLevelWithRestore(t *testing.T) {

	trySetLevelWithRestore := func() {
		defer logger.SetLogLevelWithRestore(logger.LogLevelTrace)()
		logger.Trace("You SHOULD see this trace")
	}
	trySetLevelWithRestore()
	logger.Trace("You should NOT see this trace")
}

func loggerHelperWithSkipStackFrames(skipStackFrames int, msg string) error {
	logger.Log(skipStackFrames, logger.LogLevelTrace, "myStunningPrefix:", msg)
	return nil
}

func Test_BasicUsage_SkipStackFrames(t *testing.T) {

	logger.SetLogLevel(logger.LogLevelTrace)

	// [logger_test.loggerHelperWithSkipStackFrames:...]: myStunningPrefix: hello
	_ = loggerHelperWithSkipStackFrames(0, "hello")

	// logger_test.Test_SkipStackFrames:...]: myStunningPrefix: hello
	_ = loggerHelperWithSkipStackFrames(1, "hello")
}

func Test_BasicUsage_CustomPrintLine(t *testing.T) {

	require := require.New(t)

	// Define myPrintLine
	myPrintLine := func(level logger.TLogLevel, line string) {
		line += "myPrintLine"
		logger.DefaultPrintLine(level, line)
	}

	// Use myPrintLine as logger.PrintLine

	logger.PrintLine = myPrintLine
	defer func() {
		logger.PrintLine = logger.DefaultPrintLine
	}()

	{
		logger.SetLogLevel(logger.LogLevelTrace)
		strStdout, strStderr, err := testingu.CaptureStdoutStderr(func() error {
			logger.Trace("hello")
			return nil
		})
		require.NoError(err)
		require.Empty(strStderr)
		require.Contains(strStdout, "myPrintLine")
	}

}

func Test_SkipStackFrames(t *testing.T) {

	require := require.New(t)
	logger.SetLogLevel(logger.LogLevelTrace)

	const funcNamePattern = "loggerHelperWithSkipStackFrames"

	{
		// [logger_test.loggerHelperSkip0StackFrames:69]: myStunningPrefix: hello
		strStdout, strStderr, err := testingu.CaptureStdoutStderr(func() error {
			return loggerHelperWithSkipStackFrames(0, "hello")
		})
		require.NoError(err)
		require.Empty(strStderr)
		require.Contains(strStdout, funcNamePattern)
	}

	{
		// logger_test.Test_SkipStackFrames:80]: myStunningPrefix: hello
		strStdout, strStderr, err := testingu.CaptureStdoutStderr(func() error {
			return loggerHelperWithSkipStackFrames(1, "hello")
		})
		require.NoError(err)
		require.Empty(strStderr)
		require.NotContains(strStdout, funcNamePattern)
	}

}

func Test_StdoutStderr_LogLevel(t *testing.T) {

	require := require.New(t)

	// LogLevelError
	{
		logger.SetLogLevel(logger.LogLevelError)
		strStdout, strStderr, err := testingu.CaptureStdoutStderr(func() error {
			logger.Error("Error", "arg1", "arg2")
			logger.Warning("My warning")
			return nil
		})
		require.NoError(err)
		require.Contains(strStderr, "Error arg1 arg2")
		require.Empty(strStdout)
	}

	// LogLevelWarning
	{
		logger.SetLogLevel(logger.LogLevelWarning)
		strStdout, strStderr, err := testingu.CaptureStdoutStderr(func() error {
			logger.Error("Error", "arg1", "arg2")
			logger.Warning("My warning")
			return nil
		})
		require.NoError(err)
		require.Contains(strStderr, "Error arg1 arg2")
		require.Contains(strStdout, "My warning")
	}

}

func Test_CheckSetLevels(t *testing.T) {

	require := require.New(t)

	logger.SetLogLevel(logger.LogLevelNone)
	require.False(logger.IsError())
	require.False(logger.IsWarning())
	require.False(logger.IsInfo())
	require.False(logger.IsVerbose())
	require.False(logger.IsTrace())

	logger.SetLogLevel(logger.LogLevelError)
	require.True(logger.IsError())
	require.False(logger.IsWarning())
	require.False(logger.IsInfo())
	require.False(logger.IsVerbose())
	require.False(logger.IsTrace())

	logger.SetLogLevel(logger.LogLevelWarning)
	require.True(logger.IsError())
	require.True(logger.IsWarning())
	require.False(logger.IsInfo())
	require.False(logger.IsVerbose())
	require.False(logger.IsTrace())

	logger.SetLogLevel(logger.LogLevelInfo)
	require.True(logger.IsError())
	require.True(logger.IsWarning())
	require.True(logger.IsInfo())
	require.False(logger.IsVerbose())
	require.False(logger.IsTrace())

	logger.SetLogLevel(logger.LogLevelVerbose)
	require.True(logger.IsError())
	require.True(logger.IsWarning())
	require.True(logger.IsInfo())
	require.True(logger.IsVerbose())
	require.False(logger.IsTrace())

	logger.SetLogLevel(logger.LogLevelTrace)
	require.True(logger.IsError())
	require.True(logger.IsWarning())
	require.True(logger.IsInfo())
	require.True(logger.IsVerbose())
	require.True(logger.IsTrace())

}

type mystruct struct {
}

func (m *mystruct) iWantToLog() {
	logger.Error("OOPS")
}

func TestMultithread(t *testing.T) {
	require := require.New(t)
	r, w, err := os.Pipe()
	require.NoError(err)
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = w
	wg := sync.WaitGroup{}
	wg.Add(1000)

	toLog := []string{}
	for i := 0; i < 100; i++ {
		toLog = append(toLog, strings.Repeat(strconv.Itoa(i), 10))
	}

	for i := 0; i < 1000; i++ {
		go func() {
			for i := 0; i < 100; i++ {
				logger.Info(toLog[i])
			}
			wg.Done()
		}()
	}

	stdout := ""
	wait := make(chan struct{})
	go func() {
		buf := bytes.NewBuffer(nil)
		_, err := io.Copy(buf, r)
		require.NoError(err)
		stdout = buf.String()
		close(wait)
	}()
	wg.Wait()
	w.Close()
	<-wait

	logged := strings.Split(stdout, "\n")
outer:
	for _, loggedActual := range logged {
		if len(loggedActual) == 0 {
			continue
		}
		for _, loggedExpected := range toLog {
			if strings.Contains(loggedActual, loggedExpected) {
				continue outer
			}
		}
		t.Fatal(loggedActual)
	}
}
