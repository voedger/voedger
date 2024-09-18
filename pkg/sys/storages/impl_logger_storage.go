/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

type loggerStorage struct {
}

func NewLoggerStorage() state.IStateStorage {
	return &loggerStorage{}
}

type loggerStorageKeyBuilder struct {
	baseKeyBuilder
	logLevel logger.TLogLevel
}

func (b *loggerStorageKeyBuilder) PutInt32(name string, value int32) {
	switch name {
	case sys.Storage_Logger_Field_LogLevel:
		if value < int32(logger.LogLevelNone) || value > int32(logger.LogLevelTrace) {
			panic("Invalid log level")
		}
		b.logLevel = logger.TLogLevel(value)
	default:
		b.baseKeyBuilder.PutInt32(name, value)
	}
}
func (b *loggerStorageKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*loggerStorageKeyBuilder)
	if !ok {
		return false
	}
	return b.logLevel == src.(*loggerStorageKeyBuilder).logLevel
}

type loggerStorageValueBuilder struct {
	baseValueBuilder
	message string
}

func (b *loggerStorageValueBuilder) PutString(name string, value string) {
	switch name {
	case sys.Storage_Logger_Field_Message:
		b.message = value
	default:
		b.baseValueBuilder.PutString(name, value)
	}
}

func (b *loggerStorageValueBuilder) BuildValue() istructs.IStateValue {
	return &loggerStorageValue{
		message: b.message,
	}
}

func (s *loggerStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &loggerStorageKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_Logger},
	}
}

func (s *loggerStorage) Validate([]state.ApplyBatchItem) (err error) {
	return nil
}

func (s *loggerStorage) ApplyBatch(items []state.ApplyBatchItem) (err error) {
	for _, item := range items {
		key := item.Key.(*loggerStorageKeyBuilder)
		if key.logLevel == logger.LogLevelNone {
			continue
		}
		value := item.Value.(*loggerStorageValueBuilder)
		switch key.logLevel {
		case logger.LogLevelError:
			logger.Error(value.message)
		case logger.LogLevelWarning:
			logger.Warning(value.message)
		case logger.LogLevelInfo:
			logger.Info(value.message)
		case logger.LogLevelVerbose:
			logger.Verbose(value.message)
		case logger.LogLevelTrace:
			logger.Trace(value.message)
		}
	}
	return nil
}

func (s *loggerStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &loggerStorageValueBuilder{
		baseValueBuilder{},
		"",
	}, nil
}

type loggerStorageValue struct {
	baseStateValue
	message string
}

func (v *loggerStorageValue) AsString(name string) string {
	if name == sys.Storage_Logger_Field_Message {
		return v.message
	}
	return v.baseStateValue.AsString(name)
}
