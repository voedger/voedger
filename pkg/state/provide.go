/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

func ProvideCommandProcessorStateFactory() CommandProcessorStateFactory {
	return implProvideCommandProcessorState
}
func ProvideSyncActualizerStateFactory() SyncActualizerStateFactory {
	return implProvideSyncActualizerState
}
func ProvideQueryProcessorStateFactory() QueryProcessorStateFactory {
	return implProvideQueryProcessorState
}
func ProvideAsyncActualizerStateFactory() AsyncActualizerStateFactory {
	return implProvideAsyncActualizerState
}
