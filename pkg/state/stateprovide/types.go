package stateprovide

import (
	"container/list"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
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

type wsTypeVailidator struct {
	appStructsFunc AppStructsFunc
	wsidKinds      map[wsTypeKey]appdef.QName
}

func newWsTypeValidator(appStructsFunc AppStructsFunc) wsTypeVailidator {
	return wsTypeVailidator{
		appStructsFunc: appStructsFunc,
		wsidKinds:      make(map[wsTypeKey]appdef.QName),
	}
}

// Returns NullQName if not found
func (v *wsTypeVailidator) getWSIDKind(wsid istructs.WSID, entity appdef.QName) (appdef.QName, error) {
	key := wsTypeKey{wsid: wsid, appQName: v.appStructsFunc().AppQName()}
	wsKind, ok := v.wsidKinds[key]
	if !ok {
		wsDesc, err := v.appStructsFunc().Records().GetSingleton(wsid, qNameCDocWorkspaceDescriptor)
		if err != nil {
			// notest
			return appdef.NullQName, err
		}
		if wsDesc.QName() == appdef.NullQName {
			if v.appStructsFunc().AppDef().WorkspaceByDescriptor(entity) != nil {
				// Special case. sys.CreateWorkspace creates WSKind while WorkspaceDescriptor is not applied yet.
				return entity, nil
			}
			return appdef.NullQName, fmt.Errorf("%w: %d", errWorkspaceDescriptorNotFound, wsid)
		}
		wsKind = wsDesc.AsQName(field_WSKind)
		if len(v.wsidKinds) < wsidTypeValidatorCacheSize {
			v.wsidKinds[key] = wsKind
		}
	}
	return wsKind, nil
}

func (v *wsTypeVailidator) validate(wsid istructs.WSID, entity appdef.QName) error {
	if entity == qNameCDocWorkspaceDescriptor {
		return nil // This QName always can be read and write. Otherwise sys.CreateWorkspace is not able to create descriptor.
	}
	if wsid != istructs.NullWSID && v.appStructsFunc().Records() != nil { // NullWSID only stores actualizer offsets
		wsKind, err := v.getWSIDKind(wsid, entity)
		if err != nil {
			// notest
			return err
		}
		ws := v.appStructsFunc().AppDef().WorkspaceByDescriptor(wsKind)
		if ws == nil {
			// notest
			return errDescriptorForUndefinedWorkspace
		}
		if ws.TypeByName(entity) == nil {
			return typeIsNotDefinedInWorkspaceWithDescriptor(entity, wsKind)
		}
	}
	return nil
}
