/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Final types with states:
//	- IProjector
type IWithStates interface {
	// Returns states.
	//
	// State is a storage to get data.
	//
	// States storages enumerated in alphabetical QNames order.
	// Names slice in every intent storage is sorted and deduplicated.
	States(func(storage QName, names QNames))
}

// # Final types with intents:
//	- IProjector
type IWithIntents interface {
	// Returns intents.
	//
	// Intent is a storage to put data.
	//
	// Intents storages enumerated in alphabetical QNames order.
	// Names slice in every intent storage is sorted and deduplicated.
	Intents(func(storage QName, names QNames))
}
