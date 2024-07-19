/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type hostState struct {
	istructs.IPkgNameResolver
	appStructsFunc AppStructsFunc
	name           string
	storages       map[appdef.QName]IStateStorage
	withGet        map[appdef.QName]IWithGet
	withGetBatch   map[appdef.QName]IWithGetBatch
	withRead       map[appdef.QName]IWithRead
	withApplyBatch map[appdef.QName]IWithApplyBatch
	withInsert     map[appdef.QName]IWithInsert
	withUpdate     map[appdef.QName]IWithUpdate
	intents        map[appdef.QName][]ApplyBatchItem
	intentsLimit   int
}

func newHostState(name string, intentsLimit int, appStructsFunc AppStructsFunc) *hostState {
	return &hostState{
		name:           name,
		storages:       make(map[appdef.QName]IStateStorage),
		withGet:        make(map[appdef.QName]IWithGet),
		withGetBatch:   make(map[appdef.QName]IWithGetBatch),
		withRead:       make(map[appdef.QName]IWithRead),
		withApplyBatch: make(map[appdef.QName]IWithApplyBatch),
		withInsert:     make(map[appdef.QName]IWithInsert),
		withUpdate:     make(map[appdef.QName]IWithUpdate),
		intents:        make(map[appdef.QName][]ApplyBatchItem),
		intentsLimit:   intentsLimit,
		appStructsFunc: appStructsFunc,
	}
}

func supports(ops int, op int) bool {
	return ops&op == op
}

func (s hostState) App() appdef.AppQName {
	return s.appStructsFunc().AppQName()
}

func (s hostState) AppStructs() istructs.IAppStructs {
	return s.appStructsFunc()
}

func (s hostState) CommandPrepareArgs() istructs.CommandPrepareArgs {
	panic(errCommandPrepareArgsNotSupportedByState)
}

func (s hostState) PackageFullPath(localName string) string {
	return s.appStructsFunc().AppDef().PackageFullPath(localName)
}

func (s hostState) PackageLocalName(fullPath string) string {
	return s.appStructsFunc().AppDef().PackageLocalName(fullPath)
}

func (s hostState) PLogEvent() istructs.IPLogEvent {
	panic("PLogEvent only available in actualizers")
}

func (s hostState) QueryPrepareArgs() istructs.PrepareArgs {
	panic(errQueryPrepareArgsNotSupportedByState)
}

func (s hostState) QueryCallback() istructs.ExecQueryCallback {
	panic(errQueryCallbackNotSupportedByState)
}

func (s *hostState) addStorage(storageName appdef.QName, storage IStateStorage, ops int) {
	s.storages[storageName] = storage
	if supports(ops, S_GET) {
		s.withGet[storageName] = storage.(IWithGet)
	}
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

func (s *hostState) KeyBuilder(storage, entity appdef.QName) (builder istructs.IStateKeyBuilder, err error) {
	// TODO later: re-using key builders
	strg, ok := s.storages[storage]
	if !ok {
		return nil, fmt.Errorf("%s: %w", storage, ErrUnknownStorage)
	}

	return strg.NewKeyBuilder(entity, nil), nil
}
func (s *hostState) CanExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, ok bool, err error) {

	get, ok := s.withGet[key.Storage()]
	if ok {
		value, err = get.Get(key)
		return value, value != nil, err
	}

	items := []GetBatchItem{{key: key}}
	storage, ok := s.withGetBatch[key.Storage()]
	if !ok {
		return nil, false, s.errOperationNotSupported(key.Storage(), ErrGetNotSupportedByStorage)
	}
	err = storage.GetBatch(items)
	return items[0].value, items[0].value != nil, err
}
func (s *hostState) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	batches := make(map[appdef.QName][]GetBatchItem)
	for _, k := range keys {
		batches[k.Storage()] = append(batches[k.Storage()], GetBatchItem{key: k})
	}
	for sid, batch := range batches {
		getBatch, ok := s.withGetBatch[sid]
		if ok { // GetBatch supported
			err = getBatch.GetBatch(batch)
			if err != nil {
				return err
			}
		} else { // GetBatch not supported
			get, okGet := s.withGet[sid]
			if !okGet {
				return s.errOperationNotSupported(sid, ErrGetNotSupportedByStorage)
			}
			for _, item := range batch {
				item.value, err = get.Get(item.key)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, batch := range batches {
		for _, item := range batch {
			if err := callback(item.key, item.value, item.value != nil); err != nil {
				return err
			}
		}
	}

	return nil
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
	batches := make(map[appdef.QName][]GetBatchItem)
	for _, k := range keys {
		batches[k.Storage()] = append(batches[k.Storage()], GetBatchItem{key: k})
	}
	for sid, batch := range batches {
		getBatch, ok := s.withGetBatch[sid]
		if ok { // GetBatch supported
			err = getBatch.GetBatch(batch)
			if err != nil {
				return
			}
			for _, item := range batch {
				if item.value == nil {
					return s.err(item.key, ErrNotExists)
				}
			}
		} else { // GetBatch not supported
			get, okGet := s.withGet[sid]
			if !okGet {
				return s.errOperationNotSupported(sid, ErrGetNotSupportedByStorage)
			}
			for _, item := range batch {
				item.value, err = get.Get(item.key)
				if err != nil {
					return err
				}
				if item.value == nil {
					return s.err(item.key, ErrNotExists)
				}
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
	return nil
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
	batches := make(map[appdef.QName][]GetBatchItem)
	for _, k := range keys {
		batches[k.Storage()] = append(batches[k.Storage()], GetBatchItem{key: k})
	}
	for sid, batch := range batches {
		getBatch, ok := s.withGetBatch[sid]
		if ok { // GetBatch supported
			err = getBatch.GetBatch(batch)
			if err != nil {
				return
			}
			for _, item := range batch {
				if item.value != nil {
					return s.err(item.key, ErrExists)
				}
			}
		} else { // GetBatch not supported
			get, okGet := s.withGet[sid]
			if !okGet {
				return s.errOperationNotSupported(sid, ErrGetNotSupportedByStorage)
			}
			for _, item := range batch {
				item.value, err = get.Get(item.key)
				if err != nil {
					return err
				}
				if item.value != nil {
					return s.err(item.key, ErrNotExists)
				}
			}
		}
	}
	return nil
}
func (s *hostState) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	storage, ok := s.withRead[key.Storage()]
	if !ok {
		return s.errOperationNotSupported(key.Storage(), ErrReadNotSupportedByStorage)
	}
	return storage.Read(key, callback)
}
func (s *hostState) FindIntent(key istructs.IStateKeyBuilder) istructs.IStateValueBuilder {
	for _, item := range s.intents[key.Storage()] {
		if item.key.Equals(key) {
			return item.value
		}
	}
	return nil
}

func (s *hostState) FindIntentWithOpKind(key istructs.IStateKeyBuilder) (vb istructs.IStateValueBuilder, isNew bool) {
	for _, item := range s.intents[key.Storage()] {
		if item.key.Equals(key) {
			return item.value, item.isNew
		}
	}
	return nil, false
}

func (s *hostState) IntentsCount() int {
	return s.isIntentsSize()
}

func (s *hostState) Intents(iterFunc func(key istructs.IStateKeyBuilder, value istructs.IStateValueBuilder, isNew bool)) {
	for _, items := range s.intents {
		for _, item := range items {
			iterFunc(item.key, item.value, item.isNew)
		}
	}
}

func (s *hostState) NewValue(key istructs.IStateKeyBuilder) (eb istructs.IStateValueBuilder, err error) {
	storage, ok := s.withInsert[key.Storage()]
	if !ok {
		return nil, s.errOperationNotSupported(key.Storage(), ErrInsertNotSupportedByStorage)
	}

	if s.isIntentsFull() {
		return nil, s.err(key, ErrIntentsLimitExceeded)
	}

	// TODO later: implement re-using of value builders
	builder, err := storage.ProvideValueBuilder(key, nil)
	if err != nil {
		// notest
		return nil, err
	}
	s.putToIntents(key.Storage(), key, builder, true)

	return builder, nil
}
func (s *hostState) UpdateValue(key istructs.IStateKeyBuilder, existingValue istructs.IStateValue) (eb istructs.IStateValueBuilder, err error) {
	storage, ok := s.withUpdate[key.Storage()]
	if !ok {
		return nil, s.errOperationNotSupported(key.Storage(), ErrUpdateNotSupportedByStorage)
	}

	if s.isIntentsFull() {
		return nil, s.err(key, ErrIntentsLimitExceeded)
	}

	// TODO later: implement re-using of value builders
	builder, err := storage.ProvideValueBuilderForUpdate(key, existingValue, nil)
	if err != nil {
		// notest
		return nil, err
	}
	s.putToIntents(key.Storage(), key, builder, false)

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
func (s *hostState) putToIntents(storage appdef.QName, kb istructs.IStateKeyBuilder, vb istructs.IStateValueBuilder, isNew bool) {
	s.intents[storage] = append(s.intents[storage], ApplyBatchItem{key: kb, value: vb, isNew: isNew})
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
func (s *hostState) errOperationNotSupported(sid appdef.QName, err error) error {
	return fmt.Errorf("state %s, storage %s: %w", s.name, sid, err)
}
func (s *hostState) err(key istructs.IStateKeyBuilder, err error) error {
	return fmt.Errorf("state %s, key %+v: %w", s.name, key, err)
}
