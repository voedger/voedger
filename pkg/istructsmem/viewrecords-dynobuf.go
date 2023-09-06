/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */
package istructsmem

import (
	"bytes"
	"fmt"
	"io"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

// istructs.IViewRecords.Put
func (vr *appViewRecords) storeViewRecord(workspace istructs.WSID, key istructs.IKeyBuilder, value istructs.IValueBuilder) (partKey, cCols, data []byte, err error) {

	k := key.(*keyType)
	if err = k.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = vr.app.config.validators.validKey(k, false); err != nil {
		return nil, nil, nil, err
	}

	v := value.(*valueType)
	if err = v.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = vr.app.config.validators.validViewValue(v); err != nil {
		return nil, nil, nil, err
	}

	if k.viewName != v.viewName {
		return nil, nil, nil, fmt.Errorf("key and value are from different views (key view is «%v», value view is «%v»): %w", k.viewName, v.viewName, ErrWrongDefinition)
	}

	partKey, cCols = k.storeToBytes(workspace)
	data = v.storeToBytes()

	return partKey, cCols, data, nil
}

// Stores partition key to bytes. Must be called only if valid key
func (key *keyType) storeViewPartKey(ws istructs.WSID) []byte {
	/*
		bytes   len    type     desc
		0…1      2     uint16   view QNameID
		2…9      8     uint64   WSID
		10…      ~     []~      User fields
	*/

	buf := new(bytes.Buffer)

	utils.WriteUint16(buf, key.viewID)
	utils.WriteUint64(buf, uint64(ws))

	key.partRow.fieldsDef().Fields(
		func(f appdef.IField) {
			utils.SafeWriteBuf(buf, key.partRow.dyB.Get(f.Name()))
		})

	return buf.Bytes()
}

// Stores clustering columns to bytes. Must be called only if valid key
func (key *keyType) storeViewClustKey() []byte {
	buf := new(bytes.Buffer)

	key.ccolsRow.fieldsDef().Fields(
		func(f appdef.IField) {
			utils.SafeWriteBuf(buf, key.ccolsRow.dyB.Get(f.Name()))
		})

	return buf.Bytes()
}

// Loads clustering columns from buffer
func loadViewClustKey_00(key *keyType, buf *bytes.Buffer) (err error) {
	key.ccolsRow.fieldsDef().Fields(
		func(f appdef.IField) {
			if err != nil {
				return // first error is enough
			}
			if e := loadClustFieldFromBuffer_00(key, f, buf); e != nil {
				err = fmt.Errorf("unable to load clustering columns field «%s»: %w", f.Name(), e)
			}
		})

	if err != nil {
		return err
	}

	return key.ccolsRow.build()
}

// Loads view value from specified buf using specified codec.
//
// This method uses the name of the definition set by the caller (val.QName), ignoring that is read from the buffer.
func loadViewValue(val *valueType, codecVer byte, buf *bytes.Buffer) (err error) {
	if _, err = utils.ReadUInt16(buf); err != nil {
		return fmt.Errorf("error read value QNameID: %w", err)
	}
	if err = loadRowSysFields(&val.rowType, codecVer, buf); err != nil {
		return err
	}

	len := uint32(0)
	if len, err = utils.ReadUInt32(buf); err != nil {
		return fmt.Errorf("error read value dynobuffer length: %w", err)
	}
	if buf.Len() < int(len) {
		return fmt.Errorf("error read value dynobuffer, expected %d bytes, but only %d bytes is available: %w", len, buf.Len(), io.ErrUnexpectedEOF)
	}
	val.dyB.Reset(buf.Next(int(len)))

	return nil
}

// Loads from buffer row cell
func loadClustFieldFromBuffer_00(key *keyType, field appdef.IField, buf *bytes.Buffer) (err error) {
	switch field.DataKind() {
	case appdef.DataKind_int32:
		v := int32(0)
		if v, err = utils.ReadInt32(buf); err == nil {
			key.ccolsRow.PutInt32(field.Name(), v)
		}
	case appdef.DataKind_int64:
		v := int64(0)
		if v, err = utils.ReadInt64(buf); err == nil {
			key.ccolsRow.PutInt64(field.Name(), v)
		}
	case appdef.DataKind_float32:
		v := float32(0)
		if v, err = utils.ReadFloat32(buf); err == nil {
			key.ccolsRow.PutFloat32(field.Name(), v)
		}
	case appdef.DataKind_float64:
		v := float64(0)
		if v, err = utils.ReadFloat64(buf); err == nil {
			key.ccolsRow.PutFloat64(field.Name(), v)
		}
	case appdef.DataKind_QName:
		v := uint16(0)
		if v, err = utils.ReadUInt16(buf); err == nil {
			var name appdef.QName
			if name, err = key.appCfg.qNames.QName(qnames.QNameID(v)); err == nil {
				key.ccolsRow.PutQName(field.Name(), name)
			}
		}
	case appdef.DataKind_bool:
		v := false
		if v, err = utils.ReadBool(buf); err == nil {
			key.ccolsRow.PutBool(field.Name(), v)
		}
	case appdef.DataKind_RecordID:
		v := int64(istructs.NullRecordID)
		if v, err = utils.ReadInt64(buf); err == nil {
			key.ccolsRow.PutRecordID(field.Name(), istructs.RecordID(v))
		}
	case appdef.DataKind_bytes:
		key.ccolsRow.PutBytes(field.Name(), buf.Bytes())
	case appdef.DataKind_string:
		key.ccolsRow.PutString(field.Name(), buf.String())
	default:
		//no test
		err = fmt.Errorf("unable load data type «%s»: %w", field.DataKind().TrimString(), ErrWrongFieldType)
	}
	return err
}
