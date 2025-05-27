/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package recreg

import (
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/istructs"
)

type Registry struct {
	v    istructs.IViewRecords
	keys sync.Pool
}

func New(v istructs.IViewRecords) *Registry {
	return &Registry{
		v: v,
		keys: sync.Pool{
			New: func() any {
				return v.KeyBuilder(sys.RecordsRegistryView.Name)
			},
		},
	}
}

func (reg *Registry) Get(ws istructs.WSID, id istructs.RecordID) (appdef.QName, istructs.Offset, error) {
	key := reg.key(id)
	defer reg.keys.Put(key)

	val, err := reg.v.Get(ws, key)
	if err != nil {
		return appdef.NullQName, istructs.NullOffset, err
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
