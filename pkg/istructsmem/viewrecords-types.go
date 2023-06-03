/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
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
	key := newKey(vr.app.config, view)
	if ok, err := key.validDefs(); !ok {
		panic(err)
	}
	return key
}

// istructs.IViewRecords.NewValueBuilder
func (vr *appViewRecords) NewValueBuilder(view appdef.QName) istructs.IValueBuilder {
	value := newValue(vr.app.config, view)
	if ok, err := value.validDefs(); !ok {
		panic(err)
	}
	return value
}

// istructs.IViewRecords.UpdateValueBuilder
func (vr *appViewRecords) UpdateValueBuilder(view appdef.QName, existing istructs.IValue) istructs.IValueBuilder {
	value := vr.NewValueBuilder(view).(*valueType)
	src := existing.(*valueType)
	if qName := src.QName(); qName != value.QName() {
		panic(fmt.Errorf("invalid existing value definition «%v»; expected «%v»: %w", qName, value.QName(), ErrWrongDefinition))
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
	if err = vr.app.config.validators.validKey(k, false); err != nil {
		return value, err
	}

	pKey, cKey := k.storeToBytes()
	pKey = utils.PrefixBytes(pKey, k.viewID, workspace)

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
		return fmt.Errorf("batch read %d records requested, but only %d supported: %w", len(kv), maxGetBatchRecordCount, ErrMaxGetBatchRecordCountExceeds)
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
		if err = vr.app.config.validators.validKey(k, false); err != nil {
			return fmt.Errorf("not valid key at batch item %d: %w", i, err)
		}
		pKey, cKey := k.storeToBytes()
		pKey = utils.PrefixBytes(pKey, k.viewID, workspace)
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

// istructs.IViewRecords.Read
func (vr *appViewRecords) Read(ctx context.Context, workspace istructs.WSID, key istructs.IKeyBuilder, cb istructs.ValuesCallback) (err error) {

	k := key.(*keyType)
	if err = k.build(); err != nil {
		return err
	}
	if err = vr.app.config.validators.validKey(k, true); err != nil {
		return err
	}

	pKey, cKey := k.storeToBytes()

	readRecord := func(ccols, value []byte) (err error) {
		recKey := newKey(k.appCfg, k.viewName)
		if err := recKey.loadFromBytes(pKey, ccols); err != nil {
			return err
		}

		var keyRow istructs.IRowReader
		if keyRow, err = recKey.keyRow(); err != nil {
			return err
		}

		valRow := newValue(k.appCfg, k.viewName)
		if err := valRow.loadFromBytes(value); err != nil {
			return err
		}
		return cb(keyRow, valRow)
	}

	pk := utils.PrefixBytes(pKey, k.viewID, workspace)
	return vr.app.config.storage.Read(ctx, pk, cKey, utils.IncBytes(cKey), readRecord)
}

// keyType is complex key from two parts (partition key and clustering key)
//   - interfaces:
//     — IKeyBuilder
type keyType struct {
	rowType
	viewName appdef.QName
	viewID   qnames.QNameID
	partRow  rowType
	ccolsRow rowType
}

func newKey(appCfg *AppConfigType, name appdef.QName) *keyType {
	key := keyType{
		rowType:  newRow(appCfg),
		viewName: name,
		partRow:  newRow(appCfg),
		ccolsRow: newRow(appCfg),
	}
	key.rowType.setQName(key.fkDef())
	key.partRow.setQName(key.pkDef())
	key.ccolsRow.setQName(key.ccDef())
	return &key
}

// Builds partition and clustering columns rows and returns error if occurs
func (key *keyType) build() (err error) {
	changes := key.rowType.dyB.IsModified()
	if _, err = key.rowType.build(); err != nil {
		return err
	}
	if changes {
		key.splitRow()
	}

	if _, err = key.partRow.build(); err != nil {
		return err
	}

	if _, err = key.ccolsRow.build(); err != nil {
		return err
	}

	return nil
}

// Returns name of clustering columns key definition
func (key *keyType) ccDef() appdef.QName {
	if v := key.appCfg.AppDef.View(key.viewName); v != nil {
		return v.Key().ClustCols().QName()
	}
	return appdef.NullQName
}

// Returns name of full key definition
func (key *keyType) fkDef() appdef.QName {
	if v := key.appCfg.AppDef.View(key.viewName); v != nil {
		return v.Key().QName()
	}
	return appdef.NullQName
}

// Return new key row, contained all fields from partition key and clustering columns
func (key *keyType) keyRow() (istructs.IRowReader, error) {
	row := newRow(key.appCfg)
	row.setQName(key.fkDef())

	key.partRow.fieldsDef().Fields(
		func(f appdef.IField) {
			row.dyB.Set(f.Name(), key.partRow.dyB.Get(f.Name()))
		})
	key.ccolsRow.fieldsDef().Fields(
		func(f appdef.IField) {
			row.dyB.Set(f.Name(), key.ccolsRow.dyB.Get(f.Name()))
		})

	if _, err := row.build(); err != nil {
		return nil, err
	}

	return &row, nil
}

// loadFromBytes reads key from partition key bytes and clustering columns bytes
func (key *keyType) loadFromBytes(pKey, cKey []byte) (err error) {
	buf := bytes.NewBuffer(pKey)
	if err = loadViewPartKey_00(key, buf); err != nil {
		return err
	}

	buf = bytes.NewBuffer(cKey)
	if err = loadViewClustKey_00(key, buf); err != nil {
		return err
	}

	return nil
}

// Returns name of partition key definition
func (key *keyType) pkDef() appdef.QName {
	if v := key.appCfg.AppDef.View(key.viewName); v != nil {
		return v.Key().PartKey().QName()
	}

	return appdef.NullQName
}

// Splits solid key row to partition key row and clustering columns row using view definitions
func (key *keyType) splitRow() {
	v := key.appCfg.AppDef.View(key.viewName)
	pkDef := v.Key().PartKey()
	ccDef := v.Key().ClustCols()

	key.rowType.dyB.IterateFields(nil,
		func(name string, data interface{}) bool {
			if pkDef.Field(name) != nil {
				key.partRow.dyB.Set(name, data)
			}
			if ccDef.Field(name) != nil {
				key.ccolsRow.dyB.Set(name, data)
			}
			return true
		})
}

// Stores key to partition key bytes and to clustering columns bytes
func (key *keyType) storeToBytes() (pKey, cKey []byte) {
	return key.storeViewPartKey(), key.storeViewClustKey()
}

// Checks what key has correct view, partition and clustering columns names and returns error if not
func (key *keyType) validDefs() (ok bool, err error) {
	if key.viewName == appdef.NullQName {
		return false, fmt.Errorf("missed view definition: %w", ErrNameMissed)
	}

	if key.viewID, err = key.appCfg.qNames.ID(key.viewName); err != nil {
		return false, err
	}

	d := key.appCfg.AppDef.DefByName(key.viewName)
	if d == nil {
		return false, fmt.Errorf("unknown view key definition «%v»: %w", key.viewName, ErrNameNotFound)
	}
	if d.Kind() != appdef.DefKind_ViewRecord {
		return false, fmt.Errorf("invalid view key definition «%v» kind: %w", key.viewName, ErrUnexpectedDefKind)
	}

	return true, nil
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

					equalVal := func(d1, d2 interface{}) bool {
						switch v := d1.(type) {
						case []byte: // non comparable type
							return bytes.Equal(d2.([]byte), v)
						default: // comparable types: int32, int64, float32, float64, string, bool
							return d2 == v
						}
					}

					result := true
					r1.fieldsDef().Fields(
						func(f appdef.IField) {
							result = result && equalVal(r1.dyB.Get(f.Name()), r2.dyB.Get(f.Name()))
						})
					return result
				}

				return equalRow(key.partRow, k.partRow) && equalRow(key.ccolsRow, k.ccolsRow)
			}
		}
	}
	return false
}

// istructs.IKeyBuilder.PartitionKey
func (key *keyType) PartitionKey() istructs.IRowWriter {
	return &key.partRow
}

// istructs.IKeyBuilder.ClusteringColumns
func (key *keyType) ClusteringColumns() istructs.IRowWriter {
	return &key.ccolsRow
}

// valueType implements IValue, IValueBuilder
type valueType struct {
	rowType
	viewName appdef.QName
}

func (val *valueType) Build() istructs.IValue {
	_, err := val.build()
	if err != nil {
		panic(err)
	}
	value := newValue(val.appCfg, val.viewName)
	value.copyFrom(&val.rowType)
	return value
}

func newValue(appCfg *AppConfigType, name appdef.QName) *valueType {
	value := valueType{
		rowType:  newRow(appCfg),
		viewName: name,
	}
	value.rowType.setQName(value.valueDef())
	return &value
}

// newNullValue return new empty (null) value. Useful as result if no view record found
func newNullValue() istructs.IValue {
	return newValue(NullAppConfig, appdef.NullQName)
}

// Loads view value from bytes
func (val *valueType) loadFromBytes(in []byte) (err error) {
	buf := bytes.NewBuffer(in)

	var codec byte
	if err = binary.Read(buf, binary.BigEndian, &codec); err != nil {
		return fmt.Errorf("error read codec version: %w", err)
	}
	switch codec {
	case codec_RawDynoBuffer, codec_RDB_1:
		if err := loadViewValue(val, codec, buf); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown codec version «%d»: %w", codec, ErrUnknownCodec)
	}

	return nil
}

// valueDef returns name of view value definition
func (val *valueType) valueDef() appdef.QName {
	if v := val.appCfg.AppDef.View(val.viewName); v != nil {
		return v.Value().QName()
	}
	return appdef.NullQName
}

// Checks what value has correct view and value definitions and returns error if not
func (val *valueType) validDefs() (ok bool, err error) {
	if val.viewName == appdef.NullQName {
		return false, fmt.Errorf("missed view definition: %w", ErrNameMissed)
	}

	d := val.appCfg.AppDef.DefByName(val.viewName)
	if d == nil {
		return false, fmt.Errorf("unknown view definition «%v»: %w", val.viewName, ErrNameNotFound)
	}
	if d.Kind() != appdef.DefKind_ViewRecord {
		return false, fmt.Errorf("invalid view definition «%v» kind: %w", val.viewName, ErrUnexpectedDefKind)
	}

	return true, nil
}
