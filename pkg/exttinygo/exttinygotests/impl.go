/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import (
	"github.com/voedger/voedger/pkg/exttinygo"
	"github.com/voedger/voedger/pkg/state/safestate"
	"github.com/voedger/voedger/pkg/state/teststate"
)

func NewTestState(processorKind int, packagePath string, createWorkspaces ...teststate.TestWorkspace) teststate.ITestState {

	ts := teststate.NewTestState(processorKind, packagePath, createWorkspaces...)
	sstate := safestate.ProvideSafeState(ts)

	exttinygo.KeyBuilder = func(storage, entity string) exttinygo.TKeyBuilder {
		return exttinygo.TKeyBuilder(sstate.KeyBuilder(storage, entity))
	}

	exttinygo.MustGetValue = func(k exttinygo.TKeyBuilder) exttinygo.TValue {
		return exttinygo.TValue(sstate.MustGetValue(safestate.TSafeKeyBuilder(k)))
	}

	exttinygo.QueryValue = func(k exttinygo.TKeyBuilder) (exttinygo.TValue, bool) {
		sv, ok := sstate.QueryValue(safestate.TSafeKeyBuilder(k))
		return exttinygo.TValue(sv), ok
	}

	exttinygo.NewValue = func(k exttinygo.TKeyBuilder) exttinygo.TIntent {
		return exttinygo.TIntent(sstate.NewValue(safestate.TSafeKeyBuilder(k)))
	}

	exttinygo.UpdateValue = func(k exttinygo.TKeyBuilder, existingValue exttinygo.TValue) exttinygo.TIntent {
		return exttinygo.TIntent(sstate.UpdateValue(safestate.TSafeKeyBuilder(k), safestate.TSafeValue(existingValue)))
	}

	exttinygo.KeyBuilderPutInt32 = func(k exttinygo.TKeyBuilder, name string, value int32) {
		sstate.KeyBuilderPutInt32(safestate.TSafeKeyBuilder(k), name, value)
	}

	// Intent
	exttinygo.IntentPutInt64 = func(v exttinygo.TIntent, name string, value int64) {
		sstate.IntentPutInt64(safestate.TSafeIntent(v), name, value)
	}

	// Value
	exttinygo.ValueAsValue = func(v exttinygo.TValue, name string) (result exttinygo.TValue) {
		return exttinygo.TValue(sstate.ValueAsValue(safestate.TSafeValue(v), name))
	}

	exttinygo.ValueLen = func(v exttinygo.TValue) int {
		return sstate.ValueLen(safestate.TSafeValue(v))
	}

	exttinygo.ValueGetAsValue = func(v exttinygo.TValue, index int) (result exttinygo.TValue) {
		return exttinygo.TValue(sstate.ValueGetAsValue(safestate.TSafeValue(v), index))
	}

	exttinygo.ValueAsInt32 = func(v exttinygo.TValue, name string) int32 {
		return sstate.ValueAsInt32(safestate.TSafeValue(v), name)
	}

	exttinygo.ValueAsInt64 = func(v exttinygo.TValue, name string) int64 {
		return sstate.ValueAsInt64(safestate.TSafeValue(v), name)
	}

	return ts
}
