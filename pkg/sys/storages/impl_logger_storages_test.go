/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func Test_LoggerStorage(t *testing.T) {
	storage := NewLoggerStorage()

	key := storage.NewKeyBuilder(sys.Storage_Logger, nil)
	key.PutInt32(sys.Storage_Logger_Field_LogLevel, int32(logger.LogLevelTrace))

	intents := storage.(state.IWithInsert)

	value, err := intents.ProvideValueBuilder(key, nil)
	require.NoError(t, err)

	value.PutString(sys.Storage_Logger_Field_Message, "Hello, World!")

	var line string
	logger.PrintLine = func(level logger.TLogLevel, message string) {
		line = message
	}

	logger.SetLogLevel(logger.LogLevelVerbose)
	defer logger.SetLogLevel(logger.LogLevelInfo)
	err = intents.ApplyBatch([]state.ApplyBatchItem{{Key: key, Value: value}})
	require.NoError(t, err)
	require.Empty(t, line)

	logger.SetLogLevel(logger.LogLevelTrace)
	err = intents.ApplyBatch([]state.ApplyBatchItem{{Key: key, Value: value}})
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(line, "Hello, World!"))
}
