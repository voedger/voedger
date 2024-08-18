/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package stateprovide

import (
	"container/list"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type bundle interface {
	put(key istructs.IStateKeyBuilder, value state.ApplyBatchItem)
	get(key istructs.IStateKeyBuilder) (value state.ApplyBatchItem, ok bool)
	containsKeysForSameEntity(key istructs.IStateKeyBuilder) bool
	values() (values []state.ApplyBatchItem)
	size() (size int)
	clear()
}

type pair struct {
	key   istructs.IStateKeyBuilder
	value state.ApplyBatchItem
}

type bundleImpl struct {
	list *list.List
}

func newBundle() bundle {
	return &bundleImpl{list: list.New()}
}

func (b *bundleImpl) put(key istructs.IStateKeyBuilder, value state.ApplyBatchItem) {
	for el := b.list.Front(); el != nil; el = el.Next() {
		if el.Value.(*pair).key.Equals(key) {
			el.Value.(*pair).value = value
			return
		}
	}
	b.list.PushBack(&pair{key: key, value: value})
}
func (b *bundleImpl) get(key istructs.IStateKeyBuilder) (value state.ApplyBatchItem, ok bool) {
	for el := b.list.Front(); el != nil; el = el.Next() {
		if el.Value.(*pair).key.Equals(key) {
			return el.Value.(*pair).value, true
		}
	}
	return emptyApplyBatchItem, false
}
func (b *bundleImpl) containsKeysForSameEntity(key istructs.IStateKeyBuilder) bool {
	var next *list.Element
	for el := b.list.Front(); el != nil; el = next {
		next = el.Next()
		if el.Value.(*pair).key.Entity() == key.Entity() {
			return true
		}
	}
	return false
}
func (b *bundleImpl) values() (values []state.ApplyBatchItem) {
	for el := b.list.Front(); el != nil; el = el.Next() {
		values = append(values, el.Value.(*pair).value)
	}
	return
}
func (b *bundleImpl) size() (size int) { return b.list.Len() }
func (b *bundleImpl) clear()           { b.list = list.New() }
