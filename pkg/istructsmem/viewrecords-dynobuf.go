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
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/schemas"
)

// istructs.IViewRecords.Put
func (vr *appViewRecordsType) storeViewRecord(workspace istructs.WSID, key istructs.IKeyBuilder, value istructs.IValueBuilder) (partKey, clustCols, data []byte, err error) {

	k := key.(*keyType)
	if err = k.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = vr.app.config.validators.validKey(k, false); err != nil {
		return nil, nil, nil, err
	}

	v := value.(*valueType)
	if _, err = v.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = vr.app.config.validators.validViewValue(v); err != nil {
		return nil, nil, nil, err
	}

	if k.viewName != v.viewName {
		return nil, nil, nil, fmt.Errorf("key and value are from different views (key view is «%v», value view is «%v»): %w", k.viewName, v.viewName, ErrWrongSchema)
	}

	partKey, clustCols = k.storeToBytes()
	partKey = utils.PrefixBytes(partKey, k.viewID, workspace)

	if data, err = v.storeToBytes(); err != nil {
		return nil, nil, nil, err
	}

	return partKey, clustCols, data, nil
}

// Stores partition key to bytes. Must be called only if valid key
func (key *keyType) storeViewPartKey() []byte {
	buf := new(bytes.Buffer)

	key.partRow.schema.EnumFields(
		func(f schemas.Field) {
			utils.SafeWriteBuf(buf, key.partRow.dyB.Get(f.Name()))
		})

	return buf.Bytes()
}

// Stores clustering columns to bytes. Must be called only if valid key
func (key *keyType) storeViewClustKey() []byte {
	buf := new(bytes.Buffer)

	key.clustRow.schema.EnumFields(
		func(f schemas.Field) {
			utils.SafeWriteBuf(buf, key.clustRow.dyB.Get(f.Name()))
		})

	return buf.Bytes()
}

// load functions are grouped by codec version. Codec version number included as part (suffix) in function name

// Loads partition key from buffer
func loadViewPartKey_00(key *keyType, buf *bytes.Buffer) (err error) {
	const errWrapPrefix = "unable to load partitition key"

	schema := key.partRow.schema

	schema.EnumFields(
		func(f schemas.Field) {
			if err != nil {
				return // first error is enough
			}
			if e := loadFixedLenCellFromBuffer_00(&key.partRow, f, key.appCfg, buf); e != nil {
				err = fmt.Errorf("%s: partition column «%s» cannot be loaded: %w", errWrapPrefix, f.Name(), e)
			}
		})

	if err != nil {
		return err
	}

	_, err = key.partRow.build()
	return err
}

// Loads clustering columns from buffer
func loadViewClustKey_00(key *keyType, buf *bytes.Buffer) (err error) {
	const errWrapPrefix = "unable to load clustering key"

	schema := key.clustRow.schema

	schema.EnumFields(
		func(f schemas.Field) {
			if err != nil {
				return // first error is enough
			}
			if e := loadCellFromBuffer_00(&key.clustRow, f, key.appCfg, buf); e != nil {
				err = fmt.Errorf("%s: partition column «%s» cannot be loaded: %w", errWrapPrefix, f.Name(), e)
			}
		})

	if err != nil {
		return err
	}

	_, err = key.clustRow.build()
	return err
}

// Loads from buffer row fixed-width field
func loadFixedLenCellFromBuffer_00(row *rowType, field schemas.Field, appCfg *AppConfigType, buf *bytes.Buffer) (err error) {
	switch field.DataKind() {
	case istructs.DataKind_int32:
		v := int32(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutInt32(field.Name(), v)
	case istructs.DataKind_int64:
		v := int64(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutInt64(field.Name(), v)
	case istructs.DataKind_float32:
		v := float32(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutFloat32(field.Name(), v)
	case istructs.DataKind_float64:
		v := float64(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutFloat64(field.Name(), v)
	case istructs.DataKind_QName:
		v := uint16(0)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		var name istructs.QName
		if name, err = appCfg.qNames.GetQName(qnames.QNameID(v)); err != nil {
			return err
		}
		row.PutQName(field.Name(), name)
	case istructs.DataKind_bool:
		v := false
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutBool(field.Name(), v)
	case istructs.DataKind_RecordID:
		v := int64(istructs.NullRecordID)
		if err := binary.Read(buf, binary.BigEndian, &v); err != nil {
			return err
		}
		row.PutRecordID(field.Name(), istructs.RecordID(v))
	default:
		return fmt.Errorf("field «%s» in row «%v» has variable length or unsupported field type «%v»: %w", field.Name(), row.QName(), field.DataKind(), ErrWrongFieldType)
	}
	return nil
}

// Loads from buffer row cell
func loadCellFromBuffer_00(row *rowType, field schemas.Field, appCfg *AppConfigType, buf *bytes.Buffer) (err error) {
	if field.IsFixedWidth() {
		return loadFixedLenCellFromBuffer_00(row, field, appCfg, buf)
	}
	switch field.DataKind() {
	case istructs.DataKind_bytes:
		row.PutBytes(field.Name(), buf.Bytes())
	case istructs.DataKind_string:
		row.PutString(field.Name(), buf.String())
	default:
		return fmt.Errorf("unable load data type «%v»: %w", dataKindToStr[field.DataKind()], ErrWrongFieldType)
	}
	return nil
}
