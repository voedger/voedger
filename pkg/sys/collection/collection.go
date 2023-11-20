/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

func collectionProjectorFactory(appDef appdef.IAppDef) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorCollection,
			Func: collectionProjector(appDef),
		}
	}
}

func collectionProjector(appDef appdef.IAppDef) func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		is := &idService{
			state: s,
			cache: make(map[istructs.RecordID]istructs.IRecord),
		}

		newKey := func(docQname appdef.QName, docID, elementID istructs.RecordID) (kb istructs.IStateKeyBuilder, err error) {
			kb, err = s.KeyBuilder(state.View, QNameCollectionView)
			if err != nil {
				return
			}
			kb.PutInt32(Field_PartKey, PartitionKeyCollection)
			kb.PutQName(Field_DocQName, docQname)
			kb.PutRecordID(field_DocID, docID)
			kb.PutRecordID(field_ElementID, elementID)
			return
		}

		apply := func(kb istructs.IStateKeyBuilder, record istructs.IRecord) (err error) {
			sv, ok, err := s.CanExist(kb)
			if err != nil {
				return
			}
			if ok && sv.AsInt64(state.ColOffset) >= int64(event.WLogOffset()) {
				//skip for idempotency
				return
			}
			vb, err := intents.NewValue(kb)
			if err != nil {
				return
			}
			vb.PutInt64(state.ColOffset, int64(event.WLogOffset()))
			vb.PutRecord(Field_Record, record)
			return
		}

		return event.CUDs(func(rec istructs.ICUDRow) (err error) {
			kind := appDef.Type(rec.QName()).Kind()
			if kind != appdef.TypeKind_CDoc && kind != appdef.TypeKind_CRecord {
				return
			}
			record, err := is.findRecordByID(rec.ID())
			if err != nil {
				return
			}
			root, err := is.findRootByID(rec.ID())
			if err != nil {
				return
			}
			elementID := record.ID()
			if record.ID() == root.ID() {
				elementID = istructs.NullRecordID
			}
			kb, err := newKey(root.QName(), root.ID(), elementID)
			if err != nil {
				return
			}
			return apply(kb, record)
		})
	}
}

type idService struct {
	state istructs.IState
	cache map[istructs.RecordID]istructs.IRecord
}

func (s *idService) findRecordByID(id istructs.RecordID) (record istructs.IRecord, err error) {
	record, ok := s.cache[id]
	if ok {
		return
	}

	kb, err := s.state.KeyBuilder(state.Record, appdef.NullQName)
	if err != nil {
		return
	}
	kb.PutRecordID(state.Field_ID, id)

	sv, err := s.state.MustExist(kb)
	if err != nil {
		return
	}
	record = sv.AsRecord("")

	s.cache[id] = record
	return
}
func (s *idService) findRootByID(id istructs.RecordID) (record istructs.IRecord, err error) {
	record, err = s.findRecordByID(id)
	if err != nil {
		return
	}
	if record.Parent() == istructs.NullRecordID {
		return
	}
	return s.findRootByID(record.Parent())
}
