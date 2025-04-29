/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/istructs"
)

type implIIDGenerator struct {
	nextRecordID istructs.RecordID
	onNewID      func(rawID, storageID istructs.RecordID) error
}

// used in tests
func NewIDGeneratorWithHook(onNewID func(rawID, storageID istructs.RecordID) error) istructs.IIDGenerator {
	return &implIIDGenerator{
		nextRecordID: istructs.FirstUserRecordID,
		onNewID:      onNewID,
	}
}

func NewIDGenerator() istructs.IIDGenerator {
	return NewIDGeneratorWithHook(nil)
}

func (g *implIIDGenerator) NextID(rawID istructs.RecordID) (storageID istructs.RecordID, err error) {
	storageID = g.nextRecordID
	g.nextRecordID++
	if g.onNewID != nil {
		if err := g.onNewID(rawID, storageID); err != nil {
			return istructs.NullRecordID, err
		}
	}
	return storageID, nil
}

// const minClusterRecordID = (0xFFFF - 1000 + 1) * 5_000_000_000

func (g *implIIDGenerator) UpdateOnSync(syncID istructs.RecordID) {
	// if syncID < minClusterRecordID {
	// 	// syncID<322680000000000 -> consider the syncID is from an old template.
	// 	// ignore IDs from external registers
	// 	// see https://github.com/voedger/voedger/issues/688
	// 	return
	// }
	if syncID >= g.nextRecordID {
		// we do not know the order the IDs were issued for ODoc with ORecords
		// so let's bump if syncID is actually next
		g.nextRecordID = syncID + 1
	}
}
