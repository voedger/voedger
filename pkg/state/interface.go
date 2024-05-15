/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type IStateStorage interface {
	NewKeyBuilder(entity appdef.QName, existingKeyBuilder istructs.IStateKeyBuilder) (newKeyBuilder istructs.IStateKeyBuilder)
}
type IWithGet interface {
	// Get reads item from storage
	// Nil value returned when item not found
	Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error)
}
type IWithGetBatch interface {
	// GetBatch reads items from storage
	GetBatch(items []GetBatchItem) (err error)
}
type IWithRead interface {
	// Read reads items with callback. Can return many more than 1 item for the same get
	Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error)
}
type IWithApplyBatch interface {
	// Validate validates batch before store
	Validate(items []ApplyBatchItem) (err error)
	// ApplyBatch applies batch to storage
	ApplyBatch(items []ApplyBatchItem) (err error)
}
type IWithInsert interface {
	IWithApplyBatch

	// ProvideValueBuilder provides value builder. ExistingBuilder can be null
	ProvideValueBuilder(key istructs.IStateKeyBuilder, existingBuilder istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error)
}

type IWithUpdate interface {
	IWithApplyBatch

	// ProvideValueBuilderForUpdate provides value builder to update the value. ExistingBuilder can be null
	ProvideValueBuilderForUpdate(key istructs.IStateKeyBuilder, existingValue istructs.IStateValue, existingBuilder istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error)
}

type IState interface {
	istructs.IState
	istructs.IIntents
	istructs.IPkgNameResolver
}

type IHostState interface {
	IState

	// ValidateIntents validates intents
	ValidateIntents() (err error)
	// ApplyIntents applies intents to underlying storage
	ApplyIntents() (err error)
	// ClearIntents clears intents
	ClearIntents()
}

// IBundledHostState buffers changes in "bundles" when ApplyIntents is called.
// Further Read- and *Exist operations see these changes.
type IBundledHostState interface {
	IState

	// ApplyIntents validates and stores intents to bundles
	ApplyIntents() (readyToFlushBundle bool, err error)

	// FlushBundles flushes bundles to underlying storage and resets the bundles
	FlushBundles() (err error)
}
