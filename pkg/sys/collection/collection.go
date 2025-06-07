/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"github.com/voedger/voedger/pkg/sys"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type kbAndID struct {
	istructs.IStateKeyBuilder
	id    istructs.RecordID
	isNew bool
}

var collectionProjector = istructs.Projector{
	Name: QNameProjectorCollection,
	Func: func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		is := &idService{
			state: s,
			cache: make(map[istructs.RecordID]istructs.IRecord),
		}

		newKey := func(docQname appdef.QName, docID, elementID istructs.RecordID) (kb istructs.IStateKeyBuilder, err error) {
			kb, err = s.KeyBuilder(sys.Storage_View, QNameCollectionView)
			if err != nil {
				// notest
				return
			}
			kb.PutInt32(Field_PartKey, PartitionKeyCollection)
			kb.PutQName(Field_DocQName, docQname)
			kb.PutRecordID(Field_DocID, docID)
			kb.PutRecordID(field_ElementID, elementID)
			return
		}

		apply := func(kb istructs.IStateKeyBuilder, record istructs.IRecord, isNew bool) (err error) {
			if !isNew {
				sv, ok, err := s.CanExist(kb)
				if err != nil {
					// notest
					return err
				}
				if ok && sv.AsInt64(state.ColOffset) >= int64(event.WLogOffset()) {
					// skip for idempotency
					return nil
				}
			}
			vb, err := intents.NewValue(kb)
			if err != nil {
				// notest
				return err
			}
			vb.PutInt64(state.ColOffset, int64(event.WLogOffset()))
			vb.(istructs.IStateViewValueBuilder).PutRecord(Field_Record, record)
			return nil
		}

		keyBuildersAndIDs := []kbAndID{}

		for rec := range event.CUDs {
			kind := s.AppStructs().AppDef().Type(rec.QName()).Kind()
			if kind != appdef.TypeKind_CDoc && kind != appdef.TypeKind_CRecord {
				continue
			}
			kb, err := is.state.KeyBuilder(sys.Storage_Record, appdef.NullQName)
			if err != nil {
				// notest
				return err
			}
			kb.PutRecordID(sys.Storage_Record_Field_ID, rec.ID())
			keyBuildersAndIDs = append(keyBuildersAndIDs, kbAndID{
				IStateKeyBuilder: kb,
				id:               rec.ID(),
				isNew:            rec.IsNew(),
			})
		}

		keyBuilders := make([]istructs.IStateKeyBuilder, len(keyBuildersAndIDs))
		for i, kbID := range keyBuildersAndIDs {
			keyBuilders[i] = kbID.IStateKeyBuilder
		}
		err = is.state.MustExistAll(keyBuilders, func(key istructs.IKeyBuilder, sv istructs.IStateValue, ok bool) (err error) {
			record := sv.(istructs.IStateRecordValue).AsRecord()
			is.cache[record.ID()] = record
			return nil
		})
		if err != nil {
			return err
		}
		for _, kbAndID := range keyBuildersAndIDs {
			record := is.cache[kbAndID.id]
			root, err := is.findRootByID(record.ID())
			if err != nil {
				return err
			}
			elementID := record.ID()
			if record.ID() == root.ID() {
				elementID = istructs.NullRecordID
			}
			kb, err := newKey(root.QName(), root.ID(), elementID)
			if err != nil {
				// notest
				return err
			}
			if err = apply(kb, record, kbAndID.isNew); err != nil {
				return err
			}
		}
		return nil
	},
}

type idService struct {
	state istructs.IState
	cache map[istructs.RecordID]istructs.IRecord
}

func (s *idService) findRecordByID(id istructs.RecordID) (record istructs.IRecord, err error) {
	kb, err := s.state.KeyBuilder(sys.Storage_Record, appdef.NullQName)
	if err != nil {
		return nil, err
	}
	kb.PutRecordID(sys.Storage_Record_Field_ID, id)

	sv, err := s.state.MustExist(kb)
	if err != nil {
		return nil, err
	}
	return sv.(istructs.IStateRecordValue).AsRecord(), nil
}

func (s *idService) findRootByID(id istructs.RecordID) (root istructs.IRecord, err error) {
	rec := s.cache[id]
	if rec == nil {
		if rec, err = s.findRecordByID(id); err != nil {
			return nil, err
		}
		s.cache[id] = rec
	}
	if rec.Parent() == istructs.NullRecordID {
		return rec, nil
	}
	return s.findRootByID(rec.Parent())
}
