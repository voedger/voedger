/*
 * Copyright (c) 2020-present unTill Pro, Ltd. and Contributors
 * @author Maxim Geraskin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package logger

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
)

// TLogLevel s.e.
type TLogLevel int32

// Log Levels enum
const (
	LogLevelNone = TLogLevel(iota)
	LogLevelError
	LogLevelWarning
	LogLevelInfo
	LogLevelVerbose // aka Debug
	LogLevelTrace
)

func SetLogLevel(logLevel TLogLevel) (old TLogLevel) {
	old = TLogLevel(atomic.SwapInt32((*int32)(&globalLogPrinter.logLevel), int32(logLevel)))
	return
}

func SetLogLevelWithRestore(logLevel TLogLevel) (restore func()) {
	old := SetLogLevel(logLevel)
	return func() {
		SetLogLevel(old)
		Info("LogLevel restored to", old)
	}
}

func Error(args ...interface{}) {
	printIfLevel(0, LogLevelError, args...)
}

func Warning(args ...interface{}) {
	printIfLevel(0, LogLevelWarning, args...)
}

func Info(args ...interface{}) {
	printIfLevel(0, LogLevelInfo, args...)
}

func Verbose(args ...interface{}) {
	printIfLevel(0, LogLevelVerbose, args...)
}

func Trace(args ...interface{}) {
	printIfLevel(0, LogLevelTrace, args...)
}

func Log(skipStackFrames int, level TLogLevel, args ...interface{}) {
	printIfLevel(skipStackFrames, level, args...)
}

func IsError() bool {
	return isEnabled(LogLevelError)
}

func IsInfo() bool {
	return isEnabled(LogLevelInfo)
}

func IsWarning() bool {
	return isEnabled(LogLevelWarning)
}

func IsVerbose() bool {
	return isEnabled(LogLevelVerbose)
}

func IsTrace() bool {
	return isEnabled(LogLevelTrace)
}

var PrintLine func(level TLogLevel, line string) = DefaultPrintLine

func DefaultPrintLine(level TLogLevel, line string) {
	var w io.Writer
	if level == LogLevelError {
		w = os.Stderr
	} else {
		w = os.Stdout
	}
	fmt.Fprintln(w, line)
}
