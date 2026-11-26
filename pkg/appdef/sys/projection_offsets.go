/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys

import (
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// SetKeyFields identifies a projector's offset row for both view operations and state intents.
func (v *ProjectionOffsetView) SetKeyFields(key istructs.IKeyBuilder, partition istructs.PartitionID, projector appdef.QName) {
	key.PutInt32(v.Fields.Partition, int32(partition))
	key.PutQName(v.Fields.Projector, projector)
}

// Get returns the projector's saved offset, or NullOffset when its row is missing.
func (v *ProjectionOffsetView) Get(records istructs.IViewRecords, partition istructs.PartitionID, projector appdef.QName) (istructs.Offset, error) {
	key := records.KeyBuilder(v.Name)
	v.SetKeyFields(key, partition, projector)
	value, err := records.Get(istructs.NullWSID, key)
	if errors.Is(err, istructs.ErrRecordNotFound) {
		return istructs.NullOffset, nil
	}
	if err != nil {
		return istructs.NullOffset, err
	}
	return istructs.Offset(value.AsInt64(v.Fields.Offset)), nil // nolint G115
}

// Put saves the projector's offset in the supplied application's view records.
func (v *ProjectionOffsetView) Put(records istructs.IViewRecords, partition istructs.PartitionID, projector appdef.QName, offset istructs.Offset) error {
	key := records.KeyBuilder(v.Name)
	v.SetKeyFields(key, partition, projector)
	value := records.NewValueBuilder(v.Name)
	value.PutInt64(v.Fields.Offset, int64(offset)) // nolint G115
	return records.Put(istructs.NullWSID, key, value)
}
