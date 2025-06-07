/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */
package istructsmem

import (
	"bytes"
	"io"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

// istructs.IViewRecords.Put
func (vr *appViewRecords) storeViewRecord(workspace istructs.WSID, key istructs.IKeyBuilder, value istructs.IValueBuilder) (partKey, cCols, data []byte, err error) {

	k := key.(*keyType)
	if err = k.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = validateViewKey(k, false); err != nil {
		return nil, nil, nil, err
	}

	v := value.(*valueType)
	if err = v.build(); err != nil {
		return nil, nil, nil, err
	}
	if err = validateViewValue(v); err != nil {
		return nil, nil, nil, err
	}

	if k.viewName != v.viewName {
		return nil, nil, nil, ErrWrongType("key and value are from different views (key view is «%v», value view is «%v»)", k.viewName, v.viewName)
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

	for _, f := range key.partRow.fields.Fields() {
		utils.SafeWriteBuf(buf, key.partRow.dyB.Get(f.Name()))
	}

	return buf.Bytes()
}

// Stores clustering columns to bytes. Must be called only if valid key
func (key *keyType) storeViewClustKey() []byte {
	buf := new(bytes.Buffer)

	for _, f := range key.ccolsRow.fields.Fields() {
		utils.SafeWriteBuf(buf, key.ccolsRow.dyB.Get(f.Name()))
	}

	return buf.Bytes()
}

// Loads clustering columns from buffer
func loadViewClustKey_00(key *keyType, buf *bytes.Buffer) error {
	for _, f := range key.ccolsRow.fields.Fields() {
		if err := loadClustFieldFromBuffer_00(key, f, buf); err != nil {
			return enrichError(err, key.viewName, "unable to load clustering columns field «%s»", f.Name())
		}
	}
	return key.ccolsRow.build()
}

// Loads view value from specified buf using specified codec.
//
// This method uses the name of the type set by the caller (val.QName), ignoring that is read from the buffer.
func loadViewValue(val *valueType, codecVer byte, buf *bytes.Buffer) (err error) {
	if _, err = utils.ReadUInt16(buf); err != nil {
		return enrichError(err, val.viewName, "error read value QNameID")
	}
	if err = loadRowSysFields(&val.rowType, codecVer, buf); err != nil {
		return err
	}

	length := uint32(0)
	if length, err = utils.ReadUInt32(buf); err != nil {
		return enrichError(err, val.viewName, "error read value dynobuffer length")
	}
	if buf.Len() < int(length) {
		return enrichError(io.ErrUnexpectedEOF, val.viewName, "error read value dynobuffer, expected %d bytes, but only %d bytes is available", length, buf.Len())
	}
	val.dyB.Reset(buf.Next(int(length)))

	return nil
}

// Loads from buffer row cell
func loadClustFieldFromBuffer_00(key *keyType, field appdef.IField, buf *bytes.Buffer) (err error) {
	switch field.DataKind() {
	case appdef.DataKind_int8: // #3435 [~server.vsql.smallints/cmp.istructs~impl]
		v := int8(0)
		if v, err = utils.ReadInt8(buf); err == nil {
			key.ccolsRow.PutInt8(field.Name(), v)
		}
	case appdef.DataKind_int16: // #3435 [~server.vsql.smallints/cmp.istructs~impl]
		v := int16(0)
		if v, err = utils.ReadInt16(buf); err == nil {
			key.ccolsRow.PutInt16(field.Name(), v)
		}
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
			if name, err = key.appCfg.qNames.QName(v); err == nil {
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
			key.ccolsRow.PutRecordID(field.Name(), istructs.RecordID(v)) // nolint G115
		}
	case appdef.DataKind_bytes:
		key.ccolsRow.PutBytes(field.Name(), buf.Bytes())
	case appdef.DataKind_string:
		key.ccolsRow.PutString(field.Name(), buf.String())
	default:
		// no test
		err = ErrWrongFieldType("clustering columns of «%v» unable load data type «%s»", key.viewName, field.DataKind().TrimString())
	}
	return err
}
