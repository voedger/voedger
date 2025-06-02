/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package recreg

import (
	"errors"
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/istructs"
)

// Records registry. Provide access to sys.RecordsRegistry view
type Registry struct {
	v    func() istructs.IViewRecords
	keys sync.Pool
}

// Constructs new records registry.
// The v closure will be called from the Get method to access IAppStructs.ViewRecords()
func New(v func() istructs.IViewRecords) *Registry {
	return &Registry{
		v: v,
		keys: sync.Pool{
			New: func() any {
				return v().KeyBuilder(sys.RecordsRegistryView.Name)
			},
		},
	}
}

// Returns QName and WLog offset of record by record id.
//
// If id not found then returns NullQName and NullOffset.
func (reg *Registry) Get(ws istructs.WSID, id istructs.RecordID) (appdef.QName, istructs.Offset, error) {
	key := reg.key(id)
	defer reg.keys.Put(key)

	val, err := reg.v().Get(ws, key)
	if err != nil {
		if errors.Is(err, istructs.ErrRecordNotFound) {
			// id not found, returns nulls
			return appdef.NullQName, istructs.NullOffset, nil
		}
		// get from view failed, return enriched error
		return appdef.NullQName, istructs.NullOffset, fmt.Errorf("%w: ws id %d, record id %d", err, ws, id)
	}

	return val.AsQName(sys.RecordsRegistryView.Fields.QName),
		istructs.Offset(val.AsInt64(sys.RecordsRegistryView.Fields.WLogOffset)), // nolint G115
		nil
}

func (reg *Registry) key(id istructs.RecordID) istructs.IKeyBuilder {
	key := reg.keys.Get().(istructs.IKeyBuilder)
	key.PutInt64(sys.RecordsRegistryView.Fields.IDHi, sys.RecordsRegistryView.Fields.CrackID(id))
	key.PutRecordID(sys.RecordsRegistryView.Fields.ID, id)
	return key
}
