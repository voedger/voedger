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

type bundledHostState struct {
	*hostState
	bundles      map[appdef.QName]bundle
	bundlesLimit int
}

func (s *bundledHostState) CanExist(key istructs.IStateKeyBuilder) (stateValue istructs.IStateValue, ok bool, err error) {
	bundledStorage, ok := s.bundles[key.Storage()]
	if ok {
		// can be already in a bundles
		if value, ok := bundledStorage.get(key); ok {
			// TODO later: For the optimization purposes, maybe would be wise to use e.g. AsValue()
			// instead of BuildValue()
			if stateValue = value.Value.BuildValue(); stateValue != nil {
				return stateValue, true, nil
			}
		}
	}

	return s.hostState.CanExist(key)
}
func (s *bundledHostState) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	for _, k := range keys {
		value, ok, err := s.CanExist(k)
		if err != nil {
			return err
		}
		if err = callback(k, value, ok); err != nil {
			return err
		}
	}
	return
}
func (s *bundledHostState) MustExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	value, ok, err := s.CanExist(key)
	if err != nil {
		return
	}
	if !ok {
		return nil, s.err(key, ErrNotExists)
	}
	return
}
func (s *bundledHostState) MustExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	values := make([]istructs.IStateValue, len(keys))
	for i, k := range keys {
		value, err := s.MustExist(k)
		if err != nil {
			return err
		}
		values[i] = value
	}
	for i, value := range values {
		if err = callback(keys[i], value, true); err != nil {
			return err
		}
	}
	return
}
func (s *bundledHostState) MustNotExist(key istructs.IStateKeyBuilder) (err error) {
	_, ok, err := s.CanExist(key)
	if err != nil {
		return
	}
	if ok {
		return s.err(key, ErrExists)
	}
	return
}
func (s *bundledHostState) MustNotExistAll(keys []istructs.IStateKeyBuilder) (err error) {
	for _, k := range keys {
		err = s.MustNotExist(k)
		if err != nil {
			return
		}
	}
	return
}
func (s *bundledHostState) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	bundledStorage, ok := s.bundles[key.Storage()]
	if ok {
		if bundledStorage.containsKeysForSameEntity(key) {
			err = s.FlushBundles()
			if err != nil {
				return fmt.Errorf("unable to flush on read %+v: %w", key, err)
			}
		}
	}
	return s.hostState.Read(key, callback)
}
func (s *bundledHostState) ApplyIntents() (readyToFlushBundle bool, err error) {
	defer func() {
		for sid := range s.intents {
			s.intents[sid] = s.intents[sid][0:0]
		}
	}()
	for sid, intents := range s.intents {
		if len(intents) == 0 {
			continue
		}

		err = s.withApplyBatch[sid].Validate(intents)
		if err != nil {
			return false, err
		}

		for _, item := range intents {
			s.bundles[sid].put(item.Key, item)
		}
	}
	bundles := 0
	for _, b := range s.bundles {
		bundles += b.size()
	}
	return bundles >= s.bundlesLimit, nil
}
func (s *bundledHostState) FlushBundles() (err error) {
	defer func() {
		for _, b := range s.bundles {
			b.clear()
		}
	}()
	for sid, b := range s.bundles {
		err = s.withApplyBatch[sid].ApplyBatch(b.values())
		if err != nil {
			return err
		}
	}
	return
}
func (s *bundledHostState) addStorage(storageName appdef.QName, storage state.IStateStorage, ops int) {
	s.hostState.addStorage(storageName, storage, ops)
	if supports(ops, S_UPDATE) || supports(ops, S_INSERT) {
		s.bundles[storageName] = newBundle()
	}
}
