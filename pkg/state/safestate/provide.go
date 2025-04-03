/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

package safestate

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

// oldApi is optional, to allow re-usage
func Provide(state state.IState, oldAPI safe.IStateSafeAPI) safe.IStateSafeAPI {
	if oldAPI != nil {
		// reuse
		ss := oldAPI.(*safeState)
		ss.state = state
		if len(ss.keys) > 0 {
			ss.keys = make([]istructs.IKey, 0, keysCapacity)
		}
		if len(ss.keyBuilders) > 0 {
			ss.keyBuilders = make([]istructs.IStateKeyBuilder, 0, keysBuildersCapacity)
		}
		if len(ss.values) > 0 {
			ss.values = make([]istructs.IStateValue, 0, valuesCapacity)
		}
		if len(ss.valueBuilders) > 0 {
			ss.valueBuilders = make([]istructs.IStateValueBuilder, 0, valueBuildersCapacity)
		}
		return ss
	}
	return &safeState{
		state:         state,
		keyBuilders:   make([]istructs.IStateKeyBuilder, keysBuildersCapacity),
		keys:          make([]istructs.IKey, keysCapacity),
		values:        make([]istructs.IStateValue, valuesCapacity),
		valueBuilders: make([]istructs.IStateValueBuilder, valueBuildersCapacity),
	}
}
