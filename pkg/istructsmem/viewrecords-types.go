/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"bytes"
	"context"
	"fmt"

	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/schemas"
)

// appViewRecordsType access to all application views
//   - interfaces:
//     — IViewRecords
type appViewRecordsType struct {
	app *appStructsType
}

func newAppViewRecords(app *appStructsType) appViewRecordsType {
	return appViewRecordsType{
		app: app,
	}
}

// istructs.IViewRecords.KeyBuilder
func (vr *appViewRecordsType) KeyBuilder(view istructs.QName) istructs.IKeyBuilder {
	key := newKey(vr.app.config, view)
	if ok, err := key.validSchemas(); !ok {
		panic(err)
	}
	return key
}

// istructs.IViewRecords.NewValueBuilder
func (vr *appViewRecordsType) NewValueBuilder(view istructs.QName) istructs.IValueBuilder {
	value := newValue(vr.app.config, view)
	if ok, err := value.validSchemas(); !ok {
		panic(err)
	}
	return value
}

// istructs.IViewRecords.UpdateValueBuilder
func (vr *appViewRecordsType) UpdateValueBuilder(view istructs.QName, existing istructs.IValue) istructs.IValueBuilder {
	value := vr.NewValueBuilder(view).(*valueType)
	src := existing.(*valueType)
	if qName := src.QName(); qName != value.QName() {
		panic(fmt.Errorf("invalid existing value schema «%v»; expected «%v»: %w", qName, value.QName(), ErrWrongSchema))
	}
	value.copyFrom(&src.rowType)
	return value
}

// istructs.IViewRecords.Get
func (vr *appViewRecordsType) Get(workspace istructs.WSID, key istructs.IKeyBuilder) (value istructs.IValue, err error) {
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

func (vr *appViewRecordsType) GetBatch(workspace istructs.WSID, kv []istructs.ViewRecordGetBatchItem) (err error) {
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
func (vr *appViewRecordsType) Put(workspace istructs.WSID, key istructs.IKeyBuilder, value istructs.IValueBuilder) (err error) {
	var partKey, clustCols, data []byte
	if partKey, clustCols, data, err = vr.storeViewRecord(workspace, key, value); err == nil {
		return vr.app.config.storage.Put(partKey, clustCols, data)
	}
	return err
}

// istructs.IViewRecords.PutBatch
func (vr *appViewRecordsType) PutBatch(workspace istructs.WSID, viewrecs []istructs.ViewKV) (err error) {
	batch := make([]istorage.BatchItem, len(viewrecs))

	for i, kv := range viewrecs {
		if batch[i].PKey, batch[i].CCols, batch[i].Value, err = vr.storeViewRecord(workspace, kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return vr.app.config.storage.PutBatch(batch)
}

// istructs.IViewRecords.Read
func (vr *appViewRecordsType) Read(ctx context.Context, workspace istructs.WSID, key istructs.IKeyBuilder, cb istructs.ValuesCallback) (err error) {

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
	return vr.app.config.storage.Read(ctx, pk, cKey, utils.SuccBytes(cKey), readRecord)
}

// keyType is complex key from two parts (partition key and clustering key)
//   - interfaces:
//     — IKeyBuilder
type keyType struct {
	rowType
	viewName istructs.QName
	viewID   qnames.QNameID
	partRow  rowType
	clustRow rowType
}

func newKey(appCfg *AppConfigType, name istructs.QName) *keyType {
	key := keyType{
		rowType:  newRow(appCfg),
		viewName: name,
		partRow:  newRow(appCfg),
		clustRow: newRow(appCfg),
	}
	key.rowType.setQName(key.fullKeySchema())
	key.partRow.setQName(key.partKeySchema())
	key.clustRow.setQName(key.clustColsSchema())
	return &key
}

// build builds partition and clustering columns rows and returns error if occurs
func (key *keyType) build() (err error) {
	changes := key.rowType.dyB.IsModified()
	if err = key.rowType.build(); err != nil {
		return err
	}
	if changes {
		key.splitRow()
	}

	if err = key.partRow.build(); err != nil {
		return err
	}

	if err = key.clustRow.build(); err != nil {
		return err
	}

	return nil
}

// clustColsSchema returns name of clustering columns key schema
func (key *keyType) clustColsSchema() istructs.QName {
	if schema := key.appCfg.Schemas.SchemaByName(key.viewName); schema != nil {
		if schema.Kind() == istructs.SchemaKind_ViewRecord {
			// There are no invalid schemas in the cache. Therefore, if schema is found and good kind, then no more checks necessary
			return schema.Container(istructs.SystemContainer_ViewClusteringCols).Schema()
		}
	}
	return istructs.NullQName
}

// fullKeySchema returns name of full key schema
func (key *keyType) fullKeySchema() istructs.QName {
	if schema := key.appCfg.Schemas.SchemaByName(key.viewName); schema != nil {
		if schema.Kind() == istructs.SchemaKind_ViewRecord {
			// There are no invalid schemas in the cache. Therefore, if schema is found and good kind, then no more checks necessary
			return schemas.ViewFullKeyColumsSchemaName(key.viewName)
		}
	}
	return istructs.NullQName
}

// keyRow return new key row, contained all fields from partitional key and clustering columns
func (key *keyType) keyRow() (istructs.IRowReader, error) {
	row := newRow(key.appCfg)
	row.setQName(key.fullKeySchema())

	key.partRow.schema.EnumFields(
		func(f schemas.Field) {
			row.dyB.Set(f.Name(), key.partRow.dyB.Get(f.Name()))
		})
	key.clustRow.schema.EnumFields(
		func(f schemas.Field) {
			row.dyB.Set(f.Name(), key.clustRow.dyB.Get(f.Name()))
		})

	if err := row.build(); err != nil {
		return nil, err
	}

	return &row, nil
}

// loadFromBytes reads key from partitional key bytes and clustering columns bytes
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

// partKeySchema returns name of partitional key schema
func (key *keyType) partKeySchema() istructs.QName {
	if schema := key.appCfg.Schemas.SchemaByName(key.viewName); schema != nil {
		if schema.Kind() == istructs.SchemaKind_ViewRecord {
			// There are no invalid schemas in the cache. Therefore, if schema is found and good kind, then no more checks necessary
			return schema.Container(istructs.SystemContainer_ViewPartitionKey).Schema()
		}
	}
	return istructs.NullQName
}

// splitRow splits solid key row to partition key row and clustering columns row using view schemas
func (key *keyType) splitRow() {
	partSchema := key.appCfg.Schemas.SchemaByName(key.partKeySchema())
	clustSchema := key.appCfg.Schemas.SchemaByName(key.clustColsSchema())

	key.rowType.dyB.IterateFields(nil,
		func(name string, data interface{}) bool {
			if partSchema.Field(name) != nil {
				key.partRow.dyB.Set(name, data)
			}
			if clustSchema.Field(name) != nil {
				key.clustRow.dyB.Set(name, data)
			}
			return true
		})
}

// storeToBytes stores key to partitional key bytes and to clustering columns bytes
func (key *keyType) storeToBytes() (pKey, cKey []byte) {
	return key.storeViewPartKey(), key.storeViewClustKey()
}

// validSchemas checks what key has correct view, partition and clustering columns names and returns error if not
func (key *keyType) validSchemas() (ok bool, err error) {
	if key.viewName == istructs.NullQName {
		return false, fmt.Errorf("missed view schema: %w", ErrNameMissed)
	}

	if key.viewID, err = key.appCfg.qNames.GetID(key.viewName); err != nil {
		return false, err
	}

	schema := key.appCfg.Schemas.SchemaByName(key.viewName)
	if schema == nil {
		return false, fmt.Errorf("unknown view key schema «%v»: %w", key.viewName, ErrNameNotFound)
	}
	if schema.Kind() != istructs.SchemaKind_ViewRecord {
		return false, fmt.Errorf("invalid view key schema «%v» kind: %w", key.viewName, ErrUnexpectedShemaKind)
	}

	// There are no invalid schemas in the cache. Therefore, if schema is found and good kind, then all is ok.

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
						case []byte: // uncomparable type
							return bytes.Equal(d2.([]byte), v)
						default: // comparable types: int32, int64, float32, float64, string, bool
							return d2 == v
						}
					}

					result := true
					r1.schema.EnumFields(
						func(f schemas.Field) {
							result = result && equalVal(r1.dyB.Get(f.Name()), r2.dyB.Get(f.Name()))
						})
					return result
				}

				return equalRow(key.partRow, k.partRow) && equalRow(key.clustRow, k.clustRow)
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
	return &key.clustRow
}

// valueType implements IValue, IValueBuilder
type valueType struct {
	rowType
	viewName istructs.QName
}

func (val *valueType) Build() istructs.IValue {
	err := val.build()
	if err != nil {
		panic(err)
	}
	value := newValue(val.appCfg, val.viewName)
	value.copyFrom(&val.rowType)
	return value
}

func newValue(appCfg *AppConfigType, name istructs.QName) *valueType {
	value := valueType{
		rowType:  newRow(appCfg),
		viewName: name,
	}
	value.rowType.setQName(value.valueSchema())
	return &value
}

// newNullValue return new empty (null) value. Useful as result if no view record found
func newNullValue() istructs.IValue {
	return newValue(NullAppConfig, istructs.NullQName)
}

// valueSchema returns name of view value schema
func (val *valueType) valueSchema() istructs.QName {
	if schema := val.appCfg.Schemas.SchemaByName(val.viewName); schema != nil {
		if schema.Kind() == istructs.SchemaKind_ViewRecord {
			// There are no invalid schemas in the cache. Therefore, if schema is found and good kind, then no more checks necessary
			return schema.Container(istructs.SystemContainer_ViewValue).Schema()
		}
	}
	return istructs.NullQName
}

// validSchemas checks what value has correct view and value schemas and returns error if not
func (val *valueType) validSchemas() (ok bool, err error) {
	if val.viewName == istructs.NullQName {
		return false, fmt.Errorf("missed view schema: %w", ErrNameMissed)
	}

	schema := val.appCfg.Schemas.SchemaByName(val.viewName)
	if schema == nil {
		return false, fmt.Errorf("unknown view schema «%v»: %w", val.viewName, ErrNameNotFound)
	}
	if schema.Kind() != istructs.SchemaKind_ViewRecord {
		return false, fmt.Errorf("invalid view schema «%v» kind: %w", val.viewName, ErrUnexpectedShemaKind)
	}

	// There are no invalid schemas in the cache. Therefore, if schema is found and good kind, then all is ok.

	return true, nil
}
