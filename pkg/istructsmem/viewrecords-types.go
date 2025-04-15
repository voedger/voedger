/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

// Implements IViewRecords interface
type appViewRecords struct {
	app *appStructsType
}

func newAppViewRecords(app *appStructsType) appViewRecords {
	return appViewRecords{
		app: app,
	}
}

// istructs.IViewRecords.KeyBuilder
func (vr *appViewRecords) KeyBuilder(view appdef.QName) istructs.IKeyBuilder {
	return newKey(vr.app.config, view)
}

// istructs.IViewRecords.NewValueBuilder
func (vr *appViewRecords) NewValueBuilder(view appdef.QName) istructs.IValueBuilder {
	return newValue(vr.app.config, view)
}

// istructs.IViewRecords.UpdateValueBuilder
func (vr *appViewRecords) UpdateValueBuilder(view appdef.QName, existing istructs.IValue) istructs.IValueBuilder {
	value := vr.NewValueBuilder(view).(*valueType)
	src := existing.(*valueType)
	if qName := src.QName(); qName != value.QName() {
		panic(ErrWrongType("invalid existing value type «%v»; expected «%v»", qName, value.QName()))
	}
	value.copyFrom(&src.rowType)
	return value
}

// istructs.IViewRecords.Get
func (vr *appViewRecords) Get(workspace istructs.WSID, key istructs.IKeyBuilder) (value istructs.IValue, err error) {
	value = newNullValue()

	k := key.(*keyType)
	if err = k.build(); err != nil {
		return value, err
	}
	if err = validateViewKey(k, false); err != nil {
		return value, err
	}

	pKey, cKey := k.storeToBytes(workspace)

	data := make([]byte, 0)
	if ok, err := vr.app.config.storage.Get(pKey, cKey, &data); !ok {
		if err == nil {
			err = ErrRecordNotFound
		}
		return value, err
	}

	valRow := newValue(k.appCfg, k.viewName)
	if err = valRow.loadFromBytes(data); err == nil {
		value = valRow // all is fine
	}

	return value, err
}

// istructs.IViewRecords.GetBatch
type batchPtrType struct {
	key   *keyType
	batch *istorage.GetBatchItem
}

func (vr *appViewRecords) GetBatch(workspace istructs.WSID, kv []istructs.ViewRecordGetBatchItem) (err error) {
	if len(kv) > maxGetBatchRecordCount {
		return ErrMaxGetBatchSizeExceeds(len(kv))
	}
	batches := make([]batchPtrType, len(kv))
	plan := make(map[string][]istorage.GetBatchItem)
	for i := 0; i < len(kv); i++ {
		kv[i].Ok = false
		kv[i].Value = newNullValue()
		k := kv[i].Key.(*keyType)
		if err = k.build(); err != nil {
			return fmt.Errorf("error building key at batch item %d: %w", i, err)
		}
		if err = validateViewKey(k, false); err != nil {
			return fmt.Errorf("not valid key at batch item %d: %w", i, err)
		}
		pKey, cKey := k.storeToBytes(workspace)
		batch, ok := plan[string(pKey)]
		if !ok {
			batch = make([]istorage.GetBatchItem, 0, len(kv)) // to prevent reallocation
		}
		batch = append(batch, istorage.GetBatchItem{CCols: cKey, Data: new([]byte)})
		plan[string(pKey)] = batch
		batches[i] = batchPtrType{key: k, batch: &batch[len(batch)-1]}
	}
	for pKey, batch := range plan {
		if err = vr.app.config.storage.GetBatch([]byte(pKey), batch); err != nil {
			return err
		}
	}
	for i := 0; i < len(batches); i++ {
		b := batches[i]
		kv[i].Ok = b.batch.Ok
		if kv[i].Ok {
			val := newValue(b.key.appCfg, b.key.viewName)
			if err = val.loadFromBytes(*b.batch.Data); err != nil {
				return err
			}
			kv[i].Value = val
		}
	}
	return nil
}

// istructs.IViewRecords.Put
func (vr *appViewRecords) Put(workspace istructs.WSID, key istructs.IKeyBuilder, value istructs.IValueBuilder) (err error) {
	var partKey, ccolsCols, data []byte
	if partKey, ccolsCols, data, err = vr.storeViewRecord(workspace, key, value); err == nil {
		return vr.app.config.storage.Put(partKey, ccolsCols, data)
	}
	return err
}

// istructs.IViewRecords.PutBatch
func (vr *appViewRecords) PutBatch(workspace istructs.WSID, recs []istructs.ViewKV) (err error) {
	batch := make([]istorage.BatchItem, len(recs))

	for i, kv := range recs {
		if batch[i].PKey, batch[i].CCols, batch[i].Value, err = vr.storeViewRecord(workspace, kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return vr.app.config.storage.PutBatch(batch)
}

func (vr *appViewRecords) PutJSON(ws istructs.WSID, j map[appdef.FieldName]any) error {
	viewName := appdef.NullQName

	if v, ok := j[appdef.SystemField_QName]; ok {
		if value, ok := v.(string); ok {
			if qName, err := appdef.ParseQName(value); err == nil {
				viewName = qName
			} else {
				return enrichError(err, "can not parse value for field «%s»", appdef.SystemField_QName)
			}
		} else {
			return ErrWrongFieldType("can not put «%T» to field «%s»", v, appdef.SystemField_QName)
		}
	}

	if viewName == appdef.NullQName {
		return ErrFieldIsEmpty(appdef.SystemField_QName)
	}

	view := appdef.View(vr.app.config.AppDef.Type, viewName)
	if view == nil {
		return ErrViewNotFound(viewName)
	}

	key := newKey(vr.app.config, viewName)
	value := newValue(vr.app.config, viewName)

	keyJ := make(map[appdef.FieldName]any)
	valueJ := make(map[appdef.FieldName]any)

	for f, v := range j {
		if view.Key().Field(f) != nil {
			keyJ[f] = v
		} else {
			if view.Value().Field(f) != nil {
				valueJ[f] = v
			} else {
				return ErrFieldNotFound(f, view)
			}
		}
	}

	key.PutFromJSON(keyJ)
	if err := key.build(); err != nil {
		return err
	}

	value.PutFromJSON(valueJ)
	if err := value.build(); err != nil {
		return err
	}

	return vr.Put(ws, key, value)
}

// istructs.IViewRecords.Read
func (vr *appViewRecords) Read(ctx context.Context, workspace istructs.WSID, key istructs.IKeyBuilder, cb istructs.ValuesCallback) (err error) {

	k := key.(*keyType)
	if err = k.build(); err != nil {
		return err
	}
	if err = validateViewKey(k, true); err != nil {
		return err
	}

	readRecord := func(ccols, value []byte) (err error) {
		recKey := newKey(k.appCfg, k.viewName)
		recKey.partRow.copyFrom(&k.partRow)
		if err := recKey.loadFromBytes(ccols); err != nil {
			return err
		}

		valRow := newValue(k.appCfg, k.viewName)
		if err := valRow.loadFromBytes(value); err != nil {
			return err
		}
		return cb(recKey, valRow)
	}

	pKey, cKey := k.storeToBytes(workspace)
	return vr.app.config.storage.Read(ctx, pKey, cKey, utils.IncBytes(cKey), readRecord)
}

// keyType is complex key from two parts (partition key and clustering key)
//
// # Implements:
//   - IKeyBuilder & IRowWriter
//   - IKey & IRowReader
type keyType struct {
	appCfg   *AppConfigType
	viewName appdef.QName
	viewID   istructs.QNameID
	view     appdef.IView
	partRow  rowType
	ccolsRow rowType
}

// Returns new key for specified view.
//
// # Panics:
//   - if view name is empty,
//   - if view not found
func newKey(appCfg *AppConfigType, name appdef.QName) *keyType {
	if name == appdef.NullQName {
		panic(ErrNameMissedError)
	}
	view := appdef.View(appCfg.AppDef.Type, name)
	if view == nil {
		panic(ErrViewNotFound(name))
	}
	id, err := appCfg.qNames.ID(name)
	if err != nil {
		panic(err)
	}

	key := keyType{
		appCfg:   appCfg,
		viewName: name,
		viewID:   id,
		view:     view,
		partRow:  makeRow(appCfg),
		ccolsRow: makeRow(appCfg),
	}
	key.partRow.setViewPartKey(view)
	key.ccolsRow.setViewClustCols(view)
	return &key
}

// Builds partition and clustering columns rows and returns error if occurs
func (key *keyType) build() (err error) {
	return errors.Join(
		key.partRow.build(),
		key.ccolsRow.build(),
	)
}

// Reads key from clustering columns bytes. Partition part of key must be filled (or copied) from key builder
func (key *keyType) loadFromBytes(cKey []byte) (err error) {
	buf := bytes.NewBuffer(cKey)
	if err = loadViewClustKey_00(key, buf); err != nil {
		return err
	}

	return nil
}

// Stores key to partition key bytes and to clustering columns bytes
func (key *keyType) storeToBytes(ws istructs.WSID) (pKey, cKey []byte) {
	return key.storeViewPartKey(ws), key.storeViewClustKey()
}

// istructs.IRowReader.AsBool
func (key *keyType) AsBool(name appdef.FieldName) bool {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsBool(name)
	}
	return key.ccolsRow.AsBool(name)
}

// istructs.IRowReader.AsBytes
func (key *keyType) AsBytes(name appdef.FieldName) []byte {
	return key.ccolsRow.AsBytes(name)
}

// istructs.IRowReader.AsFloat32
func (key *keyType) AsFloat32(name appdef.FieldName) float32 {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsFloat32(name)
	}
	return key.ccolsRow.AsFloat32(name)
}

// istructs.IRowReader.AsFloat64
func (key *keyType) AsFloat64(name appdef.FieldName) float64 {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsFloat64(name)
	}
	return key.ccolsRow.AsFloat64(name)
}

// #3435 [~server.vsql.smallints/cmp.istructs~impl]
//
// istructs.IRowReader.AsInt8
func (key *keyType) AsInt8(name appdef.FieldName) int8 {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsInt8(name)
	}
	return key.ccolsRow.AsInt8(name)
}

// #3435 [~server.vsql.smallints/cmp.istructs~impl]
//
// istructs.IRowReader.AsInt16
func (key *keyType) AsInt16(name appdef.FieldName) int16 {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsInt16(name)
	}
	return key.ccolsRow.AsInt16(name)
}

// istructs.IRowReader.AsInt32
func (key *keyType) AsInt32(name appdef.FieldName) int32 {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsInt32(name)
	}
	return key.ccolsRow.AsInt32(name)
}

// istructs.IRowReader.AsInt64
func (key *keyType) AsInt64(name appdef.FieldName) int64 {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsInt64(name)
	}
	return key.ccolsRow.AsInt64(name)
}

// istructs.IRowReader.AsQName
func (key *keyType) AsQName(name appdef.FieldName) appdef.QName {
	if name == appdef.SystemField_QName {
		// special case: «sys.QName» field must return view name
		return key.viewName
	}
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsQName(name)
	}
	return key.ccolsRow.AsQName(name)
}

// istructs.IRowReader.AsRecordID
func (key *keyType) AsRecordID(name appdef.FieldName) istructs.RecordID {
	if key.partRow.fieldDef(name) != nil {
		return key.partRow.AsRecordID(name)
	}
	return key.ccolsRow.AsRecordID(name)
}

// istructs.IRowReader.AsString
func (key *keyType) AsString(name appdef.FieldName) string {
	return key.ccolsRow.AsString(name)
}

// istructs.IKeyBuilder.ClusteringColumns
func (key *keyType) ClusteringColumns() istructs.IRowWriter {
	return &key.ccolsRow
}

// istructs.IKeyBuilder.Equals
func (key *keyType) Equals(src istructs.IKeyBuilder) bool {
	if k, ok := src.(*keyType); ok {
		if key == k {
			return true // same key pointer
		}
		if key.viewName != k.viewName {
			return false
		}
		if err := key.build(); err == nil {
			if err := k.build(); err == nil {

				equalRow := func(r1, r2 rowType) bool {
					if r1.dyB.Scheme != r2.dyB.Scheme {
						// notest: key.viewName == k.viewName
						return false
					}

					equalVal := func(d1, d2 interface{}) bool {
						switch v := d1.(type) {
						case []byte: // non comparable type
							return bytes.Equal(d2.([]byte), v)
						default: // comparable types: int32, int64, float32, float64, string, bool
							return d2 == v
						}
					}

					// where are no system fields in key, so we can fast iterate over dynobuffer fields
					for _, f := range r1.dyB.Scheme.Fields {
						if !equalVal(r1.dyB.GetByField(f), r2.dyB.GetByField(f)) {
							return false
						}
					}
					return true
				}

				return equalRow(key.partRow, k.partRow) && equalRow(key.ccolsRow, k.ccolsRow)
			}
		}
	}
	return false
}

// istructs.IRowReader.Fields
func (key *keyType) Fields(cb func(appdef.IField) bool) {
	key.partRow.Fields(cb)
	key.ccolsRow.Fields(cb)
}

// istructs.IRowReader.SpecifiedValues
func (key *keyType) SpecifiedValues(cb func(appdef.IField, any) bool) {
	key.partRow.SpecifiedValues(cb)
	key.ccolsRow.SpecifiedValues(cb)
}

// istructs.IKeyBuilder.PartitionKey
func (key *keyType) PartitionKey() istructs.IRowWriter {
	return &key.partRow
}

// istructs.IRowWriter.PutBool
func (key *keyType) PutBool(name appdef.FieldName, value bool) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutBool(name, value)
	} else {
		key.ccolsRow.PutBool(name, value)
	}
}

// istructs.IRowWriter.PutBytes
func (key *keyType) PutBytes(name appdef.FieldName, value []byte) {
	key.ccolsRow.PutBytes(name, value)
}

// istructs.IRowWriter.PutChars
func (key *keyType) PutChars(name appdef.FieldName, value string) {
	key.ccolsRow.PutChars(name, value)
}

// istructs.IRowWriter.PutFloat32
func (key *keyType) PutFloat32(name appdef.FieldName, value float32) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutFloat32(name, value)
	} else {
		key.ccolsRow.PutFloat32(name, value)
	}
}

// istructs.IRowWriter.PutFloat64
func (key *keyType) PutFloat64(name appdef.FieldName, value float64) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutFloat64(name, value)
	} else {
		key.ccolsRow.PutFloat64(name, value)
	}
}

// istructs.IRowWriter.PutFromJSON
func (key *keyType) PutFromJSON(j map[appdef.FieldName]any) {
	pkJ := make(map[appdef.FieldName]any)
	ccJ := make(map[appdef.FieldName]any)

	for f, v := range j {
		if key.view.Key().PartKey().Field(f) != nil {
			pkJ[f] = v
		} else if key.view.Key().ClustCols().Field(f) != nil {
			ccJ[f] = v
		}
	}

	key.partRow.PutFromJSON(pkJ)
	key.ccolsRow.PutFromJSON(ccJ)
}

// #3435 [~server.vsql.smallints/cmp.istructs~impl]
//
// istructs.IRowWriter.PutInt8
func (key *keyType) PutInt8(name appdef.FieldName, value int8) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutInt8(name, value)
	} else {
		key.ccolsRow.PutInt8(name, value)
	}
}

// #3435 [~server.vsql.smallints/cmp.istructs~impl]
//
// istructs.IRowWriter.PutInt16
func (key *keyType) PutInt16(name appdef.FieldName, value int16) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutInt16(name, value)
	} else {
		key.ccolsRow.PutInt16(name, value)
	}
}

// istructs.IRowWriter.PutInt32
func (key *keyType) PutInt32(name appdef.FieldName, value int32) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutInt32(name, value)
	} else {
		key.ccolsRow.PutInt32(name, value)
	}
}

// istructs.IRowWriter.PutInt64
func (key *keyType) PutInt64(name appdef.FieldName, value int64) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutInt64(name, value)
	} else {
		key.ccolsRow.PutInt64(name, value)
	}
}

// istructs.IRowWriter.PutNumber
func (key *keyType) PutNumber(name appdef.FieldName, value json.Number) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutNumber(name, value)
	} else {
		key.ccolsRow.PutNumber(name, value)
	}
}

// istructs.IRowWriter.PutQName
func (key *keyType) PutQName(name appdef.FieldName, value appdef.QName) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutQName(name, value)
	} else {
		key.ccolsRow.PutQName(name, value)
	}
}

// istructs.IRowWriter.PutRecordID
func (key *keyType) PutRecordID(name appdef.FieldName, value istructs.RecordID) {
	if key.partRow.fieldDef(name) != nil {
		key.partRow.PutRecordID(name, value)
	} else {
		key.ccolsRow.PutRecordID(name, value)
	}
}

// istructs.IRowWriter.PutString
func (key *keyType) PutString(name appdef.FieldName, value string) {
	key.ccolsRow.PutString(name, value)
}

// istructs.IRowReader.RecordIDs
func (key *keyType) RecordIDs(includeNulls bool) func(cb func(appdef.FieldName, istructs.RecordID) bool) {
	return func(cb func(appdef.FieldName, istructs.RecordID) bool) {
		for n, id := range key.partRow.RecordIDs(includeNulls) {
			if !cb(n, id) {
				return
			}
		}
		for n, id := range key.ccolsRow.RecordIDs(includeNulls) {
			if !cb(n, id) {
				return
			}
		}
	}
}

func (key *keyType) String() string {
	return fmt.Sprintf("view «%v» key", key.viewName)
}

// istructs.IKeyBuilder.ToBytes
func (key *keyType) ToBytes(ws istructs.WSID) (pk, cc []byte, err error) {
	if err := key.build(); err != nil {
		return nil, nil, err
	}

	pk, cc = key.storeToBytes(ws)
	return pk, cc, nil
}

// valueType implements IValue, IValueBuilder
type valueType struct {
	rowType
	viewName appdef.QName
}

// Returns new value for specified view.
//
// # Panics:
//   - if view name is empty,
//   - if view not found
func newValue(appCfg *AppConfigType, name appdef.QName) *valueType {
	if name == appdef.NullQName {
		panic(ErrNameMissedError)
	}
	view := appdef.View(appCfg.AppDef.Type, name)
	if view == nil {
		panic(ErrViewNotFound(name))
	}

	value := valueType{
		rowType:  makeRow(appCfg),
		viewName: name,
	}
	value.rowType.setType(view)
	return &value
}

// newNullValue return new empty (null) value. Useful as result if no view record found
func newNullValue() istructs.IValue {
	return &valueType{
		rowType:  makeRow(NullAppConfig),
		viewName: appdef.NullQName,
	}
}

// Loads view value from bytes
func (val *valueType) loadFromBytes(in []byte) (err error) {
	buf := bytes.NewBuffer(in)

	var codec byte
	if codec, err = utils.ReadByte(buf); err != nil {
		return fmt.Errorf("error read codec version: %w", err)
	}
	switch codec {
	case codec_RawDynoBuffer, codec_RDB_1, codec_RDB_2:
		if err := loadViewValue(val, codec, buf); err != nil {
			return err
		}
	default:
		return ErrUnknownCodec(codec)
	}

	return nil
}

func (val *valueType) Build() istructs.IValue {
	if err := val.build(); err != nil {
		panic(err)
	}
	value := newValue(val.appCfg, val.viewName)
	value.copyFrom(&val.rowType)
	return value
}

func (val *valueType) ToBytes() ([]byte, error) {
	if err := val.build(); err != nil {
		return nil, err
	}
	return val.storeToBytes(), nil
}
