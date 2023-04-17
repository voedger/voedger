/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */
package istructsmem

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// istructs.IViewRecords.Put
func (vr *appViewRecordsType) storeViewRecord(workspace istructs.WSID, key istructs.IKeyBuilder, value istructs.IValueBuilder) (partKey, clustCols, data []byte, err error) {

	k := key.(*keyType)
	if err = k.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = vr.app.config.Schemas.validKey(k, false); err != nil {
		return nil, nil, nil, err
	}

	v := value.(*valueType)
	if err = v.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = vr.app.config.Schemas.validViewValue(v); err != nil {
		return nil, nil, nil, err
	}

	if k.viewName != v.viewName {
		return nil, nil, nil, fmt.Errorf("key and value are from different views (key view is «%v», value view is «%v»): %w", k.viewName, v.viewName, ErrWrongSchema)
	}

	partKey, clustCols = k.storeToBytes()
	partKey = prefixBytes(partKey, uint16(k.viewID), uint64(workspace))

	if data, err = v.storeToBytes(); err != nil {
		return nil, nil, nil, err
	}

	return partKey, clustCols, data, nil
}

// storeViewPartKey stores partition key to bytes.
// Be careful! This method must be called only after key validation!
func (key *keyType) storeViewPartKey() []byte {
	buf := new(bytes.Buffer)
	for _, name := range key.partRow.schema.fieldsOrder {
		// if !fixedFldKind(key.partRow.schema.fields[name].kind) {…ErrFieldTypeMismatch} — unnecessary check. All view schemas must be valid

		data := key.partRow.dyB.Get(name)
		switch v := data.(type) {
		case int32, int64, float32, float64, bool:
			_ = binary.Write(buf, binary.BigEndian, v)
		case []byte: // two bytes (fld.kind == istructs.DataKind_QName)
			_, _ = buf.Write(v)
		}
		// case nil: return fmt.Errorf(… ErrNameNotFound) — unnecessary check. Key must be validated before storing
		// default: return fmt.Errorf(…, ErrFieldTypeMismatch) — unnecessary check. Key must be validated before storing
	}
	return buf.Bytes()
}

// storeViewClustKey stores clustering columns to bytes.
// Be careful! This method must be called only after key validation!
func (key *keyType) storeViewClustKey() []byte {
	buf := new(bytes.Buffer)
	for _, name := range key.clustRow.schema.fieldsOrder {
		// if (i < len(key.clustRow.schema.fieldsOrder)-1) && !fixedFldKind(key.clustRow.schema.fields[name].kind) {…ErrFieldTypeMismatch} — unnecessary check. All view schemas must be valid

		data := key.clustRow.dyB.Get(name)
		switch v := data.(type) {
		case nil:
			break // not error, just partially builded clustering key
		case int32, int64, float32, float64, bool:
			_ = binary.Write(buf, binary.BigEndian, v)
		case []byte:
			_, _ = buf.Write(v)
		case string:
			_, _ = buf.WriteString(v)
		}
		// default: return fmt.Errorf(…ErrFieldTypeMismatch) — unnecessary check. Key must be validated before storing
	}
	return buf.Bytes()
}

// load functions are grouped by codec version. Codec version number included as part (suffix) in function name

// loadViewPartKey_00 loads partition key from buffer
func loadViewPartKey_00(key *keyType, buf *bytes.Buffer) (err error) {
	const errWrapPrefix = "unable to load partitition key"

	schema := key.partRow.schema

	for _, fieldName := range schema.fieldsOrder {
		fld := schema.fields[fieldName]
		// if !fixedFldKind(fld.kind) {…ErrFieldTypeMismatch} — unnecessary check. All view schemas must be valid
		if err := loadFixedLenCellFromBuffer_00(&key.partRow, fieldName, fld, key.appCfg, buf); err != nil {
			return fmt.Errorf("%s: partition column «%s» cannot be loaded: %w", errWrapPrefix, fieldName, err)
		}
	}

	if err = key.partRow.build(); err != nil {
		return err
	}

	return nil
}

// loadViewClustKey_00 loads clustering columns from buffer
func loadViewClustKey_00(key *keyType, buf *bytes.Buffer) (err error) {
	const errWrapPrefix = "unable to load clustering key"

	schema := key.clustRow.schema

	for _, fieldName := range schema.fieldsOrder {
		fld := schema.fields[fieldName]
		// if (i < len(schema.fieldsOrder)-1) and !fixedFldKind(fld.kind) {…ErrFieldTypeMismatch)} — unnecessary check. All view schemas must be valid
		if err := loadCellFromBuffer_00(&key.clustRow, fieldName, fld, key.appCfg, buf); err != nil {
			return fmt.Errorf("%s: clustering column «%s» cannot be loaded: %w", errWrapPrefix, fieldName, err)
		}
	}

	if err = key.clustRow.build(); err != nil {
		return err
	}

	return nil
}

// loadFixedLenCellFromBuffer_00 loads from buffer row fixed-width field
func loadFixedLenCellFromBuffer_00(row *rowType, fieldName string, field *FieldPropsType, appCfg *AppConfigType, buf *bytes.Buffer) (err error) {
	switch field.kind {
	case istructs.DataKind_int32:
		v := int32(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutInt32(fieldName, v)
	case istructs.DataKind_int64:
		v := int64(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutInt64(fieldName, v)
	case istructs.DataKind_float32:
		v := float32(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutFloat32(fieldName, v)
	case istructs.DataKind_float64:
		v := float64(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutFloat64(fieldName, v)
	case istructs.DataKind_QName:
		v := uint16(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		var name istructs.QName
		if name, err = appCfg.qNames.idToQName(QNameID(v)); err != nil {
			return err
		}
		row.PutQName(fieldName, name)
	case istructs.DataKind_bool:
		v := false
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutBool(fieldName, v)
	case istructs.DataKind_RecordID:
		v := int64(istructs.NullRecordID)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutRecordID(fieldName, istructs.RecordID(v))
	default:
		return fmt.Errorf("field «%s» in row «%v» has variable length or unsupported field type «%s»: %w", fieldName, row.QName(), dataKindToStr[field.kind], coreutils.ErrFieldTypeMismatch)
	}
	return nil
}

// loadCellFromBuffer_00 loads from buffer row cell
func loadCellFromBuffer_00(row *rowType, fieldName string, field *FieldPropsType, appCfg *AppConfigType, buf *bytes.Buffer) (err error) {
	if fixedFldKind(field.kind) {
		return loadFixedLenCellFromBuffer_00(row, fieldName, field, appCfg, buf)
	}
	switch field.kind {
	case istructs.DataKind_bytes:
		row.PutBytes(fieldName, buf.Bytes())
	case istructs.DataKind_string:
		row.PutString(fieldName, buf.String())
	default:
		return fmt.Errorf("unable load data type «%s»: %w", dataKindToStr[field.kind], coreutils.ErrFieldTypeMismatch)
	}
	return nil
}

// fixedFldKind returns if fixed width field data kind
func fixedFldKind(kind istructs.DataKindType) bool {
	switch kind {
	case
		istructs.DataKind_int32,
		istructs.DataKind_int64,
		istructs.DataKind_float32,
		istructs.DataKind_float64,
		istructs.DataKind_QName,
		istructs.DataKind_bool,
		istructs.DataKind_RecordID:
		return true
	}
	return false
}
