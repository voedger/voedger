/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type implIIDGenerator struct {
	nextBaseID            istructs.RecordID
	nextCDocCRecordBaseID istructs.RecordID
	onNewID               func(rawID, storageID istructs.RecordID, def appdef.IDef) error
}

func NewIDGeneratorWithHook(onNewID func(rawID, storageID istructs.RecordID, def appdef.IDef) error) istructs.IIDGenerator {
	return &implIIDGenerator{
		nextBaseID:            istructs.FirstBaseRecordID,
		nextCDocCRecordBaseID: istructs.FirstBaseRecordID,
		onNewID:               onNewID,
	}
}

func NewIDGenerator() istructs.IIDGenerator {
	return NewIDGeneratorWithHook(nil)
}

func (g *implIIDGenerator) NextID(rawID istructs.RecordID, def appdef.IDef) (storageID istructs.RecordID, err error) {
	if def.Kind() == appdef.DefKind_CDoc || def.Kind() == appdef.DefKind_CRecord {
		storageID = istructs.NewCDocCRecordID(g.nextCDocCRecordBaseID)
		g.nextCDocCRecordBaseID++
	} else {
		storageID = istructs.NewRecordID(g.nextBaseID)
		g.nextBaseID++
	}
	if g.onNewID != nil {
		if err := g.onNewID(rawID, storageID, def); err != nil {
			return istructs.NullRecordID, err
		}
	}
	return storageID, nil
}

func (g *implIIDGenerator) UpdateOnSync(syncID istructs.RecordID, def appdef.IDef) {
	if def.Kind() == appdef.DefKind_CDoc || def.Kind() == appdef.DefKind_CRecord {
		if syncID.BaseRecordID() >= g.nextCDocCRecordBaseID {
			g.nextCDocCRecordBaseID = syncID.BaseRecordID() + 1
		}
	} else {
		if syncID.BaseRecordID() >= g.nextBaseID {
			g.nextBaseID = syncID.BaseRecordID() + 1
		}
	}
}
