/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package logger

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_MsgFormatter(t *testing.T) {
	var out string

	out = globalLogPrinter.getFormattedMsg("", "sync_op.doSync", 120, "line1")
	assert.True(t, strings.Contains(out, ": [sync_op.doSync:120]: line1"))

	out = globalLogPrinter.getFormattedMsg("", "", 121, "line1", "line2")
	assert.True(t, strings.Contains(out, ": [:121]: line1 line2"))

	out = globalLogPrinter.getFormattedMsg("m1:m2/m3", "sync_op.doSync", 126, "line1", "line2", "line3")
	assert.True(t, strings.Contains(out, "m1:m2/m3: [sync_op.doSync:126]: line1 line2 line3"))

	out = globalLogPrinter.getFormattedMsg("m1:m2/m3", "sync_op.doSync", 127, "line/1", "line/2", "line/3")
	assert.True(t, strings.Contains(out, "m1:m2/m3: [sync_op.doSync:127]: line/1 line/2 line/3"))
}

func Test_CheckRightPrefix(t *testing.T) {
	// 1. Info LogLevel = LogLevelInfo
	SetLogLevel(LogLevelInfo)
	assert.Equal(t, getLevelPrefix(globalLogPrinter.logLevel), infoPrefix)

	// 2. Trace LogLevel = LogLevelTrace
	SetLogLevel(LogLevelTrace)
	assert.Equal(t, getLevelPrefix(globalLogPrinter.logLevel), tracePrefix)

	// 3. Warning LogLevel = LogLevelWarning
	SetLogLevel(LogLevelWarning)
	assert.Equal(t, getLevelPrefix(globalLogPrinter.logLevel), warningPrefix)

	// 4. Error LogLevel = LogLevelError
	SetLogLevel(LogLevelError)
	assert.Equal(t, getLevelPrefix(globalLogPrinter.logLevel), errorPrefix)

	// 5. Unexisting level
	SetLogLevel(7)
	assert.Equal(t, getLevelPrefix(globalLogPrinter.logLevel), "")

	SetLogLevel(LogLevelInfo)
}

func Test_GetFuncName(t *testing.T) {
	funcName, line := globalLogPrinter.getFuncName(2)
	assert.Equal(t, funcName, "testing.tRunner")
	assert.True(t, line > 0)
}

func Benchmark_FuncForPC(b *testing.B) {
	var funcName string

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		funcName, _ = globalLogPrinter.getFuncName(2)
	}
	assert.Equal(b, funcName, "testing.(*B).runN")

	elapsed := time.Since(start).Seconds()
	b.ReportMetric(float64(b.N)/elapsed, "rps")
}
