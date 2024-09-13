/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type hostState struct {
	istructs.IPkgNameResolver
	appStructsFunc state.AppStructsFunc
	name           string
	storages       map[appdef.QName]state.IStateStorage
	withGet        map[appdef.QName]state.IWithGet
	withGetBatch   map[appdef.QName]state.IWithGetBatch
	withRead       map[appdef.QName]state.IWithRead
	withApplyBatch map[appdef.QName]state.IWithApplyBatch
	withInsert     map[appdef.QName]state.IWithInsert
	withUpdate     map[appdef.QName]state.IWithUpdate
	intents        map[appdef.QName][]state.ApplyBatchItem
	intentsLimit   int
}

func newHostState(name string, intentsLimit int, appStructsFunc state.AppStructsFunc) *hostState {
	return &hostState{
		name:           name,
		storages:       make(map[appdef.QName]state.IStateStorage),
		withGet:        make(map[appdef.QName]state.IWithGet),
		withGetBatch:   make(map[appdef.QName]state.IWithGetBatch),
		withRead:       make(map[appdef.QName]state.IWithRead),
		withApplyBatch: make(map[appdef.QName]state.IWithApplyBatch),
		withInsert:     make(map[appdef.QName]state.IWithInsert),
		withUpdate:     make(map[appdef.QName]state.IWithUpdate),
		intents:        make(map[appdef.QName][]state.ApplyBatchItem),
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

func (s *hostState) addStorage(storageName appdef.QName, storage state.IStateStorage, ops int) {
	s.storages[storageName] = storage
	if supports(ops, S_GET) {
		s.withGet[storageName] = storage.(state.IWithGet)
	}
	if supports(ops, S_GET_BATCH) {
		s.withGetBatch[storageName] = storage.(state.IWithGetBatch)
	}
	if supports(ops, S_READ) {
		s.withRead[storageName] = storage.(state.IWithRead)
	}
	if supports(ops, S_INSERT) {
		s.withApplyBatch[storageName] = storage.(state.IWithApplyBatch)
		s.withInsert[storageName] = storage.(state.IWithInsert)
	}
	if supports(ops, S_UPDATE) {
		s.withApplyBatch[storageName] = storage.(state.IWithApplyBatch)
		s.withUpdate[storageName] = storage.(state.IWithUpdate)
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

	items := []state.GetBatchItem{{Key: key}}
	storage, ok := s.withGetBatch[key.Storage()]
	if !ok {
		return nil, false, s.errOperationNotSupported(key.Storage(), ErrGetNotSupportedByStorage)
	}
	err = storage.GetBatch(items)
	return items[0].Value, items[0].Value != nil, err
}
func (s *hostState) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	batches := make(map[appdef.QName][]state.GetBatchItem)
	for _, k := range keys {
		batches[k.Storage()] = append(batches[k.Storage()], state.GetBatchItem{Key: k})
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
				item.Value, err = get.Get(item.Key)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, batch := range batches {
		for _, item := range batch {
			if err := callback(item.Key, item.Value, item.Value != nil); err != nil {
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
	batches := make(map[appdef.QName][]state.GetBatchItem)
	for _, k := range keys {
		batches[k.Storage()] = append(batches[k.Storage()], state.GetBatchItem{Key: k})
	}
	for sid, batch := range batches {
		getBatch, ok := s.withGetBatch[sid]
		if ok { // GetBatch supported
			err = getBatch.GetBatch(batch)
			if err != nil {
				return
			}
			for _, item := range batch {
				if item.Value == nil {
					return s.err(item.Key, ErrNotExists)
				}
			}
		} else { // GetBatch not supported
			get, okGet := s.withGet[sid]
			if !okGet {
				return s.errOperationNotSupported(sid, ErrGetNotSupportedByStorage)
			}
			for _, item := range batch {
				item.Value, err = get.Get(item.Key)
				if err != nil {
					return err
				}
				if item.Value == nil {
					return s.err(item.Key, ErrNotExists)
				}
			}
		}
	}
	for _, batch := range batches {
		for _, item := range batch {
			if err := callback(item.Key, item.Value, true); err != nil {
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
	batches := make(map[appdef.QName][]state.GetBatchItem)
	for _, k := range keys {
		batches[k.Storage()] = append(batches[k.Storage()], state.GetBatchItem{Key: k})
	}
	for sid, batch := range batches {
		getBatch, ok := s.withGetBatch[sid]
		if ok { // GetBatch supported
			err = getBatch.GetBatch(batch)
			if err != nil {
				return
			}
			for _, item := range batch {
				if item.Value != nil {
					return s.err(item.Key, ErrExists)
				}
			}
		} else { // GetBatch not supported
			get, okGet := s.withGet[sid]
			if !okGet {
				return s.errOperationNotSupported(sid, ErrGetNotSupportedByStorage)
			}
			for _, item := range batch {
				item.Value, err = get.Get(item.Key)
				if err != nil {
					return err
				}
				if item.Value != nil {
					return s.err(item.Key, ErrNotExists)
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
		if item.Key.Equals(key) {
			return item.Value
		}
	}
	return nil
}

func (s *hostState) FindIntentWithOpKind(key istructs.IStateKeyBuilder) (vb istructs.IStateValueBuilder, isNew bool) {
	for _, item := range s.intents[key.Storage()] {
		if item.Key.Equals(key) {
			return item.Value, item.IsNew
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
			iterFunc(item.Key, item.Value, item.IsNew)
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
	s.intents[storage] = append(s.intents[storage], state.ApplyBatchItem{Key: kb, Value: vb, IsNew: isNew})
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
	return fmt.Errorf("state %s, key {%s}: %w", s.name, key, err)
}
