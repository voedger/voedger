/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

type viewRecordsStorage struct {
	ctx              context.Context
	appStructsFunc   state.AppStructsFunc
	wsidFunc         state.WSIDFunc
	n10nFunc         state.N10nFunc
	wsTypeVailidator wsTypeVailidator
}

type iViewInt64FieldTypeChecker interface {
	isViewInt64FieldRecordID(name appdef.QName, fieldName appdef.FieldName) bool
}

func NewViewRecordsStorage(ctx context.Context, appStructsFunc state.AppStructsFunc, wsidFunc state.WSIDFunc, n10nFunc state.N10nFunc) state.IStateStorage {
	return &viewRecordsStorage{
		ctx:              ctx,
		appStructsFunc:   appStructsFunc,
		wsidFunc:         wsidFunc,
		n10nFunc:         n10nFunc,
		wsTypeVailidator: newWsTypeValidator(appStructsFunc),
	}
}
func (s *viewRecordsStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) (newKeyBuilder istructs.IStateKeyBuilder) {
	return &viewKeyBuilder{
		IKeyBuilder: s.appStructsFunc().ViewRecords().KeyBuilder(entity),
		view:        entity,
		wsid:        s.wsidFunc(),
	}
}

func (s *viewRecordsStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*viewKeyBuilder)
	err = s.wsTypeVailidator.validate(k.wsid, k.view)
	if err != nil {
		return nil, err
	}
	v, err := s.appStructsFunc().ViewRecords().Get(k.wsid, k.IKeyBuilder)
	if err != nil {
		if errors.Is(err, istructs.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return &viewValue{
		value: v,
	}, nil
}

func (s *viewRecordsStorage) GetBatch(items []state.GetBatchItem) (err error) {
	wsidToItemIdx := make(map[istructs.WSID][]int)
	batches := make(map[istructs.WSID][]istructs.ViewRecordGetBatchItem)
	for itemIdx, item := range items {
		k := item.Key.(*viewKeyBuilder)
		if err = s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
			return err
		}
		wsidToItemIdx[k.wsid] = append(wsidToItemIdx[k.wsid], itemIdx)
		batches[k.wsid] = append(batches[k.wsid], istructs.ViewRecordGetBatchItem{Key: k.IKeyBuilder})
	}
	for wsid, batch := range batches {
		err = s.appStructsFunc().ViewRecords().GetBatch(wsid, batch)
		if err != nil {
			return
		}
		for i, batchItem := range batch {
			itemIndex := wsidToItemIdx[wsid][i]
			if !batchItem.Ok {
				continue
			}
			items[itemIndex].Value = &viewValue{
				value: batchItem.Value,
			}
		}
	}
	return err
}
func (s *viewRecordsStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	cb := func(k istructs.IKey, v istructs.IValue) (err error) {
		return callback(k, &viewValue{
			value: v,
		})
	}
	k := kb.(*viewKeyBuilder)
	if err = s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
		return err
	}
	return s.appStructsFunc().ViewRecords().Read(s.ctx, k.wsid, k.IKeyBuilder, cb)
}
func (s *viewRecordsStorage) Validate([]state.ApplyBatchItem) (err error) { return err }
func (s *viewRecordsStorage) ApplyBatch(items []state.ApplyBatchItem) (err error) {
	batches := make(map[istructs.WSID][]istructs.ViewKV)
	nn := make(map[n10n]istructs.Offset)
	for _, item := range items {
		k := item.Key.(*viewKeyBuilder)
		v := item.Value.(*viewValueBuilder)
		batches[k.wsid] = append(batches[k.wsid], istructs.ViewKV{Key: k.IKeyBuilder, Value: v.IValueBuilder})
		if nn[n10n{wsid: k.wsid, view: k.view}] < v.offset {
			nn[n10n{wsid: k.wsid, view: k.view}] = v.offset
		}
	}
	var nullWsidBatch []istructs.ViewKV
	for wsid, batch := range batches {
		if wsid == istructs.NullWSID { // Actualizer offsets must be updated in the last order
			nullWsidBatch = batch
			continue
		}
		err = s.appStructsFunc().ViewRecords().PutBatch(wsid, batch)
		if err != nil {
			return err
		}
	}
	if len(nullWsidBatch) > 0 {
		err = s.appStructsFunc().ViewRecords().PutBatch(istructs.NullWSID, nullWsidBatch)
		if err != nil {
			return err
		}
	}
	for n, newOffset := range nn {
		if logger.IsVerbose() {
			logger.Verbose(fmt.Sprintf("viewRecordStorage: sending n10n view: %s, wsid: %d, newOffset: %d", n.view, n.wsid, newOffset))
		}
		s.n10nFunc(n.view, n.wsid, newOffset)
	}
	return err
}
func (s *viewRecordsStorage) ProvideValueBuilder(kb istructs.IStateKeyBuilder, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	k := kb.(*viewKeyBuilder)
	if err := s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
		return nil, err
	}
	return &viewValueBuilder{
		IValueBuilder: s.appStructsFunc().ViewRecords().NewValueBuilder(k.view),
		offset:        istructs.NullOffset,
		entity:        kb.Entity(),
		fc:            &s.wsTypeVailidator,
	}, nil
}
func (s *viewRecordsStorage) ProvideValueBuilderForUpdate(kb istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	k := kb.(*viewKeyBuilder)
	if err := s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
		return nil, err
	}
	return &viewValueBuilder{
		IValueBuilder: s.appStructsFunc().ViewRecords().UpdateValueBuilder(kb.(*viewKeyBuilder).view, existingValue.(*viewValue).value),
		offset:        istructs.NullOffset,
		entity:        kb.Entity(),
		fc:            &s.wsTypeVailidator,
	}, nil
}

type viewKeyBuilder struct {
	istructs.IKeyBuilder
	wsid istructs.WSID
	view appdef.QName
}

func (b *viewKeyBuilder) PutInt64(name string, value int64) {
	if name == sys.Storage_View_Field_WSID {
		wsid, err := coreutils.Int64ToWSID(value)
		if err != nil {
			panic(err)
		}
		b.wsid = wsid
		return
	}
	b.IKeyBuilder.PutInt64(name, value)
}
func (b *viewKeyBuilder) PutQName(name string, value appdef.QName) {
	if name == appdef.SystemField_QName {
		b.wsid = istructs.NullWSID
		b.view = value
	}
	b.IKeyBuilder.PutQName(name, value)
}
func (b *viewKeyBuilder) Entity() appdef.QName {
	return b.view
}
func (b *viewKeyBuilder) Storage() appdef.QName {
	return sys.Storage_View
}
func (b *viewKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*viewKeyBuilder)
	if !ok {
		return false
	}
	if b.wsid != kb.wsid {
		return false
	}
	if b.view != kb.view {
		return false
	}
	return b.IKeyBuilder.Equals(kb.IKeyBuilder)
}
func (b *viewKeyBuilder) String() string {
	bb := new(bytes.Buffer)
	fmt.Fprintf(bb, "storage:%s", b.Storage())
	fmt.Fprintf(bb, ", entity:%s", b.Entity())
	fmt.Fprintf(bb, ", wsid:%d", b.wsid)
	return bb.String()
}

type viewValueBuilder struct {
	istructs.IValueBuilder
	istructs.IStateViewValueBuilder
	offset istructs.Offset
	entity appdef.QName
	fc     iViewInt64FieldTypeChecker
}

// used in tests
func (b *viewValueBuilder) Equal(src istructs.IStateValueBuilder) bool {
	bThis, err := b.IValueBuilder.ToBytes()
	if err != nil {
		panic(err)
	}
	srcValueBuilder, ok := src.(*viewValueBuilder)
	if !ok {
		return false
	}
	bSrc, err := srcValueBuilder.IValueBuilder.ToBytes()
	if err != nil {
		panic(err)
	}

	return reflect.DeepEqual(bThis, bSrc)
}

func (b *viewValueBuilder) PutRecord(name string, record istructs.IRecord) {
	b.IValueBuilder.PutRecord(name, record)
}
func (b *viewValueBuilder) ToBytes() ([]byte, error) {
	return b.IValueBuilder.ToBytes()
}
func (b *viewValueBuilder) PutInt64(name string, value int64) {
	if name == state.ColOffset {
		b.offset = istructs.Offset(value) // nolint G115
	}
	if b.fc.isViewInt64FieldRecordID(b.entity, name) {
		recID, err := coreutils.Int64ToRecordID(value)
		if err != nil {
			panic(err)
		}
		b.IValueBuilder.PutRecordID(name, recID)
	} else {
		b.IValueBuilder.PutInt64(name, value)
	}
}
func (b *viewValueBuilder) PutQName(name string, value appdef.QName) {
	if name == appdef.SystemField_QName {
		b.offset = istructs.NullOffset
	}
	b.IValueBuilder.PutQName(name, value)
}
func (b *viewValueBuilder) Build() istructs.IValue {
	return b.IValueBuilder.Build()
}

func (b *viewValueBuilder) BuildValue() istructs.IStateValue {
	return &viewValue{
		value: b.Build(),
	}
}

type viewValue struct {
	baseStateValue
	istructs.IStateViewValue
	value istructs.IValue
}

func (v *viewValue) AsInt8(name string) int8          { return v.value.AsInt8(name) }
func (v *viewValue) AsInt16(name string) int16        { return v.value.AsInt16(name) }
func (v *viewValue) AsInt32(name string) int32        { return v.value.AsInt32(name) }
func (v *viewValue) AsInt64(name string) int64        { return v.value.AsInt64(name) }
func (v *viewValue) AsFloat32(name string) float32    { return v.value.AsFloat32(name) }
func (v *viewValue) AsFloat64(name string) float64    { return v.value.AsFloat64(name) }
func (v *viewValue) AsBytes(name string) []byte       { return v.value.AsBytes(name) }
func (v *viewValue) AsString(name string) string      { return v.value.AsString(name) }
func (v *viewValue) AsQName(name string) appdef.QName { return v.value.AsQName(name) }
func (v *viewValue) AsBool(name string) bool          { return v.value.AsBool(name) }
func (v *viewValue) AsRecordID(name string) istructs.RecordID {
	return v.value.AsRecordID(name)
}
func (v *viewValue) AsRecord(name string) istructs.IRecord {
	return v.value.AsRecord(name)
}

type n10n struct {
	wsid istructs.WSID
	view appdef.QName
}
