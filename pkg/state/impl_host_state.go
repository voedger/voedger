/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

type hostState struct {
	name           string
	storages       map[schemas.QName]IStateStorage
	withGetBatch   map[schemas.QName]IWithGetBatch
	withRead       map[schemas.QName]IWithRead
	withApplyBatch map[schemas.QName]IWithApplyBatch
	withInsert     map[schemas.QName]IWithInsert
	withUpdate     map[schemas.QName]IWithUpdate
	intents        map[schemas.QName][]ApplyBatchItem
	intentsLimit   int
}

func newHostState(name string, intentsLimit int) *hostState {
	return &hostState{
		name:           name,
		storages:       make(map[schemas.QName]IStateStorage),
		withGetBatch:   make(map[schemas.QName]IWithGetBatch),
		withRead:       make(map[schemas.QName]IWithRead),
		withApplyBatch: make(map[schemas.QName]IWithApplyBatch),
		withInsert:     make(map[schemas.QName]IWithInsert),
		withUpdate:     make(map[schemas.QName]IWithUpdate),
		intents:        make(map[schemas.QName][]ApplyBatchItem),
		intentsLimit:   intentsLimit,
	}
}

func supports(ops int, op int) bool {
	return ops&op == op
}

func (s *hostState) addStorage(storageName schemas.QName, storage IStateStorage, ops int) {
	s.storages[storageName] = storage
	if supports(ops, S_GET_BATCH) {
		s.withGetBatch[storageName] = storage.(IWithGetBatch)
	}
	if supports(ops, S_READ) {
		s.withRead[storageName] = storage.(IWithRead)
	}
	if supports(ops, S_INSERT) {
		s.withApplyBatch[storageName] = storage.(IWithApplyBatch)
		s.withInsert[storageName] = storage.(IWithInsert)
	}
	if supports(ops, S_UPDATE) {
		s.withApplyBatch[storageName] = storage.(IWithApplyBatch)
		s.withUpdate[storageName] = storage.(IWithUpdate)
	}
}

func (s *hostState) KeyBuilder(storage, entity schemas.QName) (builder istructs.IStateKeyBuilder, err error) {
	// TODO later: re-using key builders
	strg, ok := s.storages[storage]
	if !ok {
		return nil, fmt.Errorf("%s: %w", storage, ErrUnknownStorage)
	}

	return strg.NewKeyBuilder(entity, nil), nil
}
func (s *hostState) CanExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, ok bool, err error) {
	items := []GetBatchItem{{key: key}}
	storage, ok := s.withGetBatch[getStorageID(key)]
	if !ok {
		return nil, false, s.errOperationNotSupported(getStorageID(key), ErrGetBatchNotSupportedByStorage)
	}

	err = storage.GetBatch(items)
	if err != nil {
		return nil, false, err
	}
	return items[0].value, items[0].value != nil, err
}
func (s *hostState) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	batches := make(map[schemas.QName][]GetBatchItem)
	for _, k := range keys {
		batches[getStorageID(k)] = append(batches[getStorageID(k)], GetBatchItem{key: k})
	}
	for sid, batch := range batches {
		storage, ok := s.withGetBatch[sid]
		if !ok {
			return s.errOperationNotSupported(sid, ErrGetBatchNotSupportedByStorage)
		}
		err = storage.GetBatch(batch)
		if err != nil {
			return
		}
		for _, item := range batch {
			if err := callback(item.key, item.value, item.value != nil); err != nil {
				return err
			}
		}
	}
	return
}
func (s *hostState) MustExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	value, ok, err := s.CanExist(key)
	if err != nil {
		return
	}
	if !ok {
		return nil, s.err(key, ErrNotExists)
	}
	return
}
func (s *hostState) MustExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	batches := make(map[schemas.QName][]GetBatchItem)
	for _, k := range keys {
		batches[getStorageID(k)] = append(batches[getStorageID(k)], GetBatchItem{key: k})
	}
	for sid, batch := range batches {
		storage, ok := s.withGetBatch[sid]
		if !ok {
			return s.errOperationNotSupported(sid, ErrGetBatchNotSupportedByStorage)
		}
		err = storage.GetBatch(batch)
		if err != nil {
			return
		}
		for _, item := range batch {
			if item.value == nil {
				return s.err(item.key, ErrNotExists)
			}
		}
	}
	for _, batch := range batches {
		for _, item := range batch {
			if err := callback(item.key, item.value, true); err != nil {
				return err
			}
		}
	}
	return
}
func (s *hostState) MustNotExist(key istructs.IStateKeyBuilder) (err error) {
	_, ok, err := s.CanExist(key)
	if err != nil {
		return
	}
	if ok {
		return s.err(key, ErrExists)
	}
	return
}
func (s *hostState) MustNotExistAll(keys []istructs.IStateKeyBuilder) (err error) {
	batches := make(map[schemas.QName][]GetBatchItem)
	for _, k := range keys {
		batches[getStorageID(k)] = append(batches[getStorageID(k)], GetBatchItem{key: k})
	}
	for sid, batch := range batches {
		storage, ok := s.withGetBatch[sid]
		if !ok {
			return s.errOperationNotSupported(sid, ErrGetBatchNotSupportedByStorage)
		}
		err = storage.GetBatch(batch)
		if err != nil {
			return
		}
		for _, item := range batch {
			if item.value != nil {
				return s.err(item.key, ErrExists)
			}
		}
	}
	return
}
func (s *hostState) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	storage, ok := s.withRead[getStorageID(key)]
	if !ok {
		return s.errOperationNotSupported(getStorageID(key), ErrReadNotSupportedByStorage)
	}
	return storage.Read(key, callback)
}
func (s *hostState) NewValue(key istructs.IStateKeyBuilder) (eb istructs.IStateValueBuilder, err error) {
	storage, ok := s.withInsert[getStorageID(key)]
	if !ok {
		return nil, s.errOperationNotSupported(getStorageID(key), ErrInsertNotSupportedByStorage)
	}

	if s.isIntentsFull() {
		return nil, s.err(key, ErrIntentsLimitExceeded)
	}

	// TODO later: implement re-using of value builders
	builder := storage.ProvideValueBuilder(key, nil)
	s.putToIntents(getStorageID(key), key, builder)

	return builder, nil
}
func (s *hostState) UpdateValue(key istructs.IStateKeyBuilder, existingValue istructs.IStateValue) (eb istructs.IStateValueBuilder, err error) {
	storage, ok := s.withUpdate[getStorageID(key)]
	if !ok {
		return nil, s.errOperationNotSupported(getStorageID(key), ErrUpdateNotSupportedByStorage)
	}

	if s.isIntentsFull() {
		return nil, s.err(key, ErrIntentsLimitExceeded)
	}

	// TODO later: implement re-using of value builders
	builder := storage.ProvideValueBuilderForUpdate(key, existingValue, nil)
	s.putToIntents(getStorageID(key), key, builder)

	return builder, nil
}
func (s *hostState) ValidateIntents() (err error) {
	if s.isIntentsEmpty() {
		return nil
	}
	for sid, items := range s.intents {
		err = s.withApplyBatch[sid].Validate(items)
		if err != nil {
			return
		}
	}
	return
}
func (s *hostState) ApplyIntents() (err error) {
	if s.isIntentsEmpty() {
		return nil
	}
	defer func() {
		for sid := range s.intents {
			s.intents[sid] = s.intents[sid][0:0]
		}
	}()
	for sid, items := range s.intents {
		err = s.withApplyBatch[sid].ApplyBatch(items)
		if err != nil {
			return
		}
	}
	return nil
}
func (s *hostState) ClearIntents() {
	for sid := range s.intents {
		s.intents[sid] = s.intents[sid][0:0]
	}
}
func (s *hostState) putToIntents(storage schemas.QName, kb istructs.IStateKeyBuilder, vb istructs.IStateValueBuilder) {
	s.intents[storage] = append(s.intents[storage], ApplyBatchItem{key: kb, value: vb})
}
func (s *hostState) isIntentsFull() bool {
	return s.isIntentsSize() >= s.intentsLimit
}
func (s *hostState) isIntentsEmpty() bool {
	return s.isIntentsSize() == 0
}
func (s *hostState) isIntentsSize() int {
	intentsSize := 0
	for _, items := range s.intents {
		intentsSize += len(items)
	}
	return intentsSize
}
func (s *hostState) errOperationNotSupported(sid schemas.QName, err error) error {
	return fmt.Errorf("state %s, storage %s: %w", s.name, sid, err)
}
func (s *hostState) err(key istructs.IStateKeyBuilder, err error) error {
	return fmt.Errorf("state %s, key %+v: %w", s.name, key, err)
}
