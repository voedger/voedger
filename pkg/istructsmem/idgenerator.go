/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/istructs"
)

type implIIDGenerator struct {
	nextBaseID istructs.RecordID
	onNewID    func(rawID, storageID istructs.RecordID) error
}

// used in tests
func NewIDGeneratorWithHook(onNewID func(rawID, storageID istructs.RecordID) error) istructs.IIDGenerator {
	return &implIIDGenerator{
		nextBaseID: istructs.FirstBaseRecordID,
		onNewID:    onNewID,
	}
}

func NewIDGenerator() istructs.IIDGenerator {
	return NewIDGeneratorWithHook(nil)
}

func (g *implIIDGenerator) NextID(rawID istructs.RecordID) (storageID istructs.RecordID, err error) {
	storageID = istructs.NewRecordID(g.nextBaseID)
	g.nextBaseID++
	if g.onNewID != nil {
		if err := g.onNewID(rawID, storageID); err != nil {
			return istructs.NullRecordID, err
		}
	}
	return storageID, nil
}

func (g *implIIDGenerator) UpdateOnSync(syncID istructs.RecordID) {
	if syncID < istructs.MinClusterRecordID {
		// syncID<322680000000000 -> consider the syncID is from an old template.
		// ignore IDs from external registers
		// see https://github.com/voedger/voedger/issues/688
		return
	}
	if syncID.BaseRecordID() >= g.nextBaseID {
		// we do not know the order the IDs were issued for ODoc with ORecords
		// so let's bump if syncID is actually next
		g.nextBaseID = syncID.BaseRecordID() + 1
	}
}
