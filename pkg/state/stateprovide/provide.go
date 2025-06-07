/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import "github.com/voedger/voedger/pkg/state"

func ProvideCommandProcessorStateFactory() state.CommandProcessorStateFactory {
	return implProvideCommandProcessorState
}

func ProvideMockedCommandProcessorStateFactory() state.MockedStateFactory {
	return implProvideMockedCommandProcessorState
}

func ProvideMockedActualizerStateFactory() state.MockedStateFactory {
	return implProvideMockedActualizerState
}

func ProvideSyncActualizerStateFactory() state.SyncActualizerStateFactory {
	return implProvideSyncActualizerState
}

func ProvideQueryProcessorStateFactory() state.QueryProcessorStateFactory {
	return implProvideQueryProcessorState
}

func ProvideAsyncActualizerStateFactory() state.AsyncActualizerStateFactory {
	return implProvideAsyncActualizerState
}

func ProvideSchedulerStateFactory() state.SchedulerStateFactory {
	return implProvideSchedulerState
}
