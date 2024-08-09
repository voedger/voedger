/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type recordsStorage struct {
	recordsFunc      recordsFunc
	cudFunc          CUDFunc
	wsidFunc         WSIDFunc
	wsTypeVailidator wsTypeVailidator
}

type recordsKeyBuilder struct {
	baseKeyBuilder
	id          istructs.RecordID
	singleton   appdef.QName
	isSingleton bool
	wsid        istructs.WSID
}

func (b *recordsKeyBuilder) Storage() appdef.QName {
	return sys.Storage_Record
}
func (b *recordsKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*recordsKeyBuilder)
	if !ok {
		return false
	}
	if b.id != kb.id {
		return false
	}
	if b.singleton != kb.singleton {
		return false
	}
	if b.isSingleton != kb.isSingleton {
		return false
	}
	if b.wsid != kb.wsid {
		return false
	}
	return true
}
func (b *recordsKeyBuilder) String() string {
	sb := strings.Builder{}
	_, _ = sb.WriteString(fmt.Sprintf("- %T", b))
	if b.id != istructs.NullRecordID {
		_, _ = sb.WriteString(fmt.Sprintf(", ID - %d", b.id))
	}
	if b.singleton != appdef.NullQName {
		_, _ = sb.WriteString(fmt.Sprintf(", singleton - %s", b.singleton))
	}
	if b.isSingleton {
		_, _ = sb.WriteString(", singleton")
	}
	_, _ = sb.WriteString(fmt.Sprintf(", WSID - %d", b.wsid))
	return sb.String()
}
func (b *recordsKeyBuilder) PutInt64(name string, value int64) {
	if name == sys.Storage_Record_Field_WSID {
		b.wsid = istructs.WSID(value)
		return
	}
	if name == sys.Storage_Record_Field_ID {
		b.id = istructs.RecordID(value)
		return
	}
	b.baseKeyBuilder.PutInt64(name, value)
}
func (b *recordsKeyBuilder) PutRecordID(name string, value istructs.RecordID) {
	if name == sys.Storage_Record_Field_ID {
		b.id = value
		return
	}
	b.baseKeyBuilder.PutRecordID(name, value)
}
func (b *recordsKeyBuilder) PutBool(name string, value bool) {
	if name == sys.Storage_Record_Field_IsSingleton {
		if b.entity == appdef.NullQName {
			panic("entity undefined")
		}
		b.isSingleton = value
		return
	}
	b.baseKeyBuilder.PutBool(name, value)
}
func (b *recordsKeyBuilder) PutQName(name string, value appdef.QName) {
	if name == sys.Storage_Record_Field_Singleton {
		b.singleton = value
		return
	}
	b.baseKeyBuilder.PutQName(name, value)
}

func newRecordsStorage(appStructsFunc AppStructsFunc, wsidFunc WSIDFunc, cudFunc CUDFunc) *recordsStorage {
	return &recordsStorage{
		recordsFunc:      func() istructs.IRecords { return appStructsFunc().Records() },
		wsidFunc:         wsidFunc,
		cudFunc:          cudFunc,
		wsTypeVailidator: newWsTypeValidator(appStructsFunc),
	}
}

func (s *recordsStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &recordsKeyBuilder{
		id:             istructs.NullRecordID,
		singleton:      appdef.NullQName, // Deprecated, use isSingleton instead
		isSingleton:    false,
		wsid:           s.wsidFunc(),
		baseKeyBuilder: baseKeyBuilder{entity: entity},
	}
}

func (s *recordsStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*recordsKeyBuilder)
	if k.isSingleton || k.singleton != appdef.NullQName {

		qname := k.singleton // for compatibility
		if k.isSingleton {
			qname = k.entity
		}

		err = s.wsTypeVailidator.validate(k.wsid, qname)
		if err != nil {
			return nil, err
		}
		singleton, e := s.recordsFunc().GetSingleton(k.wsid, qname)
		if e != nil {
			return nil, e
		}
		if singleton.QName() == appdef.NullQName {
			return nil, nil
		}
		return &recordsValue{record: singleton}, nil
	}
	if k.id == istructs.NullRecordID {
		// error message according to https://dev.untill.com/projects/#!637229
		return nil, fmt.Errorf("value of one of RecordID fields is 0: %w", ErrNotFound)
	}
	record, err := s.recordsFunc().Get(k.wsid, true, k.id)
	if err != nil {
		return nil, err
	}
	if record.QName() == appdef.NullQName {
		return nil, nil
	}
	return &recordsValue{record: record}, nil
}

func (s *recordsStorage) GetBatch(items []GetBatchItem) (err error) {
	type getSingletonParams struct {
		wsid    istructs.WSID
		qname   appdef.QName
		itemIdx int
	}
	wsidToItemIdx := make(map[istructs.WSID][]int)
	batches := make(map[istructs.WSID][]istructs.RecordGetBatchItem)
	gg := make([]getSingletonParams, 0)
	for itemIdx, item := range items {
		k := item.key.(*recordsKeyBuilder)
		if k.isSingleton || k.singleton != appdef.NullQName {
			qname := k.singleton // for compatibility
			if k.isSingleton {
				qname = k.entity
			}
			err = s.wsTypeVailidator.validate(k.wsid, qname)
			if err != nil {
				return err
			}
			gg = append(gg, getSingletonParams{
				wsid:    k.wsid,
				qname:   qname,
				itemIdx: itemIdx,
			})
			continue
		}
		if k.id == istructs.NullRecordID {
			// error message according to https://dev.untill.com/projects/#!637229
			return fmt.Errorf("value of one of RecordID fields is 0: %w", ErrNotFound)
		}
		wsidToItemIdx[k.wsid] = append(wsidToItemIdx[k.wsid], itemIdx)
		batches[k.wsid] = append(batches[k.wsid], istructs.RecordGetBatchItem{ID: k.id})
	}
	for wsid, batch := range batches {
		err = s.recordsFunc().GetBatch(wsid, true, batch)
		if err != nil {
			return
		}
		for i, batchItem := range batch {
			if batchItem.Record.QName() == appdef.NullQName {
				continue
			}
			items[wsidToItemIdx[wsid][i]].value = &recordsValue{record: batchItem.Record}
		}
	}
	for _, g := range gg {
		singleton, e := s.recordsFunc().GetSingleton(g.wsid, g.qname)
		if e != nil {
			return e
		}
		if singleton.QName() == appdef.NullQName {
			continue
		}
		items[g.itemIdx].value = &recordsValue{record: singleton}
	}
	return err
}
func (s *recordsStorage) Validate([]ApplyBatchItem) (err error)   { return }
func (s *recordsStorage) ApplyBatch([]ApplyBatchItem) (err error) { return }
func (s *recordsStorage) ProvideValueBuilder(key istructs.IStateKeyBuilder, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	kb := key.(*recordsKeyBuilder)
	if kb.entity == appdef.NullQName {
		return nil, errEntityRequiredForValueBuilder
	}
	err := s.wsTypeVailidator.validate(kb.wsid, kb.entity)
	if err != nil {
		return nil, err
	}
	rw := s.cudFunc().Create(kb.entity)
	return &recordsValueBuilder{rw: rw}, nil
}
func (s *recordsStorage) ProvideValueBuilderForUpdate(_ istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &recordsValueBuilder{rw: s.cudFunc().Update(existingValue.AsRecord(""))}, nil
}

type recordsValueBuilder struct {
	istructs.IStateValueBuilder
	rw istructs.IRowWriter
}

func (b *recordsValueBuilder) BuildValue() istructs.IStateValue {
	rr, err := istructs.BuildRow(b.rw)
	if err != nil {
		panic(err)
	}
	if rec, ok := rr.(istructs.IRecord); ok {
		return &recordsValue{record: rec}
	}
	return nil
}

func (b *recordsValueBuilder) Equal(src istructs.IStateValueBuilder) bool {
	vb, ok := src.(*recordsValueBuilder)
	if !ok {
		return false
	}
	return reflect.DeepEqual(b.rw, vb.rw) // TODO: does that work?
}
func (b *recordsValueBuilder) PutInt32(name string, value int32)        { b.rw.PutInt32(name, value) }
func (b *recordsValueBuilder) PutInt64(name string, value int64)        { b.rw.PutInt64(name, value) }
func (b *recordsValueBuilder) PutBytes(name string, value []byte)       { b.rw.PutBytes(name, value) }
func (b *recordsValueBuilder) PutString(name, value string)             { b.rw.PutString(name, value) }
func (b *recordsValueBuilder) PutBool(name string, value bool)          { b.rw.PutBool(name, value) }
func (b *recordsValueBuilder) PutChars(name string, value string)       { b.rw.PutChars(name, value) }
func (b *recordsValueBuilder) PutFloat32(name string, value float32)    { b.rw.PutFloat32(name, value) }
func (b *recordsValueBuilder) PutFloat64(name string, value float64)    { b.rw.PutFloat64(name, value) }
func (b *recordsValueBuilder) PutQName(name string, value appdef.QName) { b.rw.PutQName(name, value) }
func (b *recordsValueBuilder) PutNumber(name string, value float64)     { b.rw.PutNumber(name, value) }
func (b *recordsValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	b.rw.PutRecordID(name, value)
}
