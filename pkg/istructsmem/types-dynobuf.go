/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/untillpro/dynobuffers"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/containers"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

// Converts specified value to the value according to data kind
// number types must be the same as DataKind, e.g. DataKind_int32 -> int32 is expected only
// float64 value is accepted in case of DataKind_float64 only
// If value type is not corresponding to kind then next conversions are available:
//
//	— json.Number can be converted to all numeric kinds (int32, int64, float32, float64, RecordID)
//	  — overflowing is checked
//	— string value can be converted to QName and []byte kinds
//
// QName values, record- and event- values returned as []byte
func (row *rowType) clarifyJSONValue(value interface{}, kind appdef.DataKind) (res interface{}, err error) {
outer:
	switch kind {
	case appdef.DataKind_int32:
		switch v := value.(type) {
		case int32:
			return v, nil
		case json.Number:
			return coreutils.ClarifyJSONNumber(v, kind)
		}
	case appdef.DataKind_int64:
		switch v := value.(type) {
		case int64:
			return v, nil
		case json.Number:
			return coreutils.ClarifyJSONNumber(v, kind)
		}
	case appdef.DataKind_float32:
		switch v := value.(type) {
		case float32:
			return v, nil
		case json.Number:
			return coreutils.ClarifyJSONNumber(v, kind)
		}
	case appdef.DataKind_float64:
		switch v := value.(type) {
		case float64:
			return v, nil
		case json.Number:
			return coreutils.ClarifyJSONNumber(v, kind)
		}
	case appdef.DataKind_RecordID:
		var int64Val int64
		switch v := value.(type) {
		case int64:
			int64Val = v
		case istructs.RecordID:
			return v, nil
		case json.Number:
			return coreutils.ClarifyJSONNumber(v, kind)
		default:
			break outer
		}
		if int64Val < 0 {
			return nil, ErrWrongRecordID("negative value %d", int64Val)
		}
		return istructs.RecordID(int64Val), nil
	case appdef.DataKind_bytes:
		switch v := value.(type) {
		case string:
			bytes, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, err
			}
			return bytes, nil
		case []byte:
			return v, nil
		}
	case appdef.DataKind_string:
		switch v := value.(type) {
		case string:
			return v, nil
		}
	case appdef.DataKind_QName:
		switch v := value.(type) {
		case string:
			qName, err := appdef.ParseQName(v)
			if err != nil {
				return nil, err
			}
			id, err := row.appCfg.qNames.ID(qName)
			if err != nil {
				return nil, err
			}
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, id)
			return b, nil
		case appdef.QName:
			id, err := row.appCfg.qNames.ID(v)
			if err != nil {
				return nil, err
			}
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, id)
			return b, nil
		}
	case appdef.DataKind_bool:
		switch v := value.(type) {
		case bool:
			return v, nil
		}
	case appdef.DataKind_Record:
		switch v := value.(type) {
		case *recordType:
			return v.storeToBytes(), nil
		}
	case appdef.DataKind_Event:
		switch v := value.(type) {
		case *eventType:
			return v.storeToBytes(), nil
		}
	}
	return nil, ErrWrongFieldType("value has type «%T», but «%s» expected", value, kind.TrimString())
}

func dynoBufGetWord(dyB *dynobuffers.Buffer, fieldName appdef.FieldName) (value uint16, ok bool) {
	if b := dyB.GetByteArray(fieldName); b != nil {
		if bytes := b.Bytes(); len(bytes) == 2 {
			value = binary.BigEndian.Uint16(bytes)
			return value, true
		}
	}
	return 0, false
}

func storeRow(row *rowType, buf *bytes.Buffer) {
	id, err := row.qNameID()
	if err != nil {
		// no test
		panic(enrichError(err, row))
	}
	utils.WriteUint16(buf, id)
	if row.QName() == appdef.NullQName {
		return
	}

	storeRowSysFields(row, buf)

	b, err := row.dyB.ToBytes()
	if err != nil {
		// no test
		panic(enrichError(err, row))
	}
	length := uint32(len(b)) // nolint G115 considering int32 is enough to store the event
	utils.WriteUint32(buf, length)
	utils.SafeWriteBuf(buf, b)
}

func storeRowSysFields(row *rowType, buf *bytes.Buffer) {
	sysFieldMask := uint16(0)
	if row.ID() != istructs.NullRecordID {
		sysFieldMask |= sfm_ID
	}
	if row.parentID != istructs.NullRecordID {
		sysFieldMask |= sfm_ParentID
	}
	if row.container != "" {
		sysFieldMask |= sfm_Container
	}
	if !row.isActive {
		sysFieldMask |= sfm_IsActive
	}

	utils.WriteUint16(buf, sysFieldMask)

	if row.ID() != istructs.NullRecordID {
		utils.WriteUint64(buf, uint64(row.ID()))
	}
	if row.parentID != istructs.NullRecordID {
		utils.WriteUint64(buf, uint64(row.parentID))
	}
	if row.container != "" {
		id, err := row.containerID()
		if err != nil {
			// no test
			panic(enrichError(err, row))
		}
		utils.WriteUint16(buf, uint16(id))
	}
	if !row.isActive {
		utils.WriteBool(buf, false)
	}
}

func loadRow(row *rowType, codecVer byte, buf *bytes.Buffer) (err error) {
	row.clear()

	var qnameId uint16
	if qnameId, err = utils.ReadUInt16(buf); err != nil {
		return fmt.Errorf("error read row QNameID: %w", err)
	}
	if err = row.setQNameID(qnameId); err != nil {
		return err
	}
	if row.QName() == appdef.NullQName {
		return nil
	}

	if err = loadRowSysFields(row, codecVer, buf); err != nil {
		return err
	}

	length := uint32(0)
	if length, err = utils.ReadUInt32(buf); err != nil {
		return fmt.Errorf("error read dynobuffer length: %w", err)
	}
	if buf.Len() < int(length) {
		return fmt.Errorf("error read dynobuffer, expected %d bytes, but only %d bytes is available: %w", length, buf.Len(), io.ErrUnexpectedEOF)
	}
	row.dyB.Reset(buf.Next(int(length)))

	return nil
}

// Returns system fields mask combination for type kind, see sfm_××× consts
func typeKindSysFieldsMask(kind appdef.TypeKind) uint16 {
	sfm := uint16(0)
	if exists, _ := kind.HasSystemField(appdef.SystemField_ID); exists {
		sfm |= sfm_ID
	}
	if exists, _ := kind.HasSystemField(appdef.SystemField_ParentID); exists {
		sfm |= sfm_ParentID
	}
	if exists, _ := kind.HasSystemField(appdef.SystemField_Container); exists {
		sfm |= sfm_Container
	}
	if exists, _ := kind.HasSystemField(appdef.SystemField_IsActive); exists {
		sfm |= sfm_IsActive
	}
	return sfm
}

func loadRowSysFields(row *rowType, codecVer byte, buf *bytes.Buffer) (err error) {
	var sysFieldMask uint16

	if codecVer == codec_RawDynoBuffer {
		sysFieldMask = typeKindSysFieldsMask(row.typ.Kind())
	} else {
		if sysFieldMask, err = utils.ReadUInt16(buf); err != nil {
			return fmt.Errorf("error read system fields mask: %w", err)
		}
	}

	if (sysFieldMask & sfm_ID) == sfm_ID {
		var id uint64
		if id, err = utils.ReadUInt64(buf); err != nil {
			return fmt.Errorf("error read record ID: %w", err)
		}
		row.setID(istructs.RecordID(id))
	}
	if (sysFieldMask & sfm_ParentID) == sfm_ParentID {
		var id uint64
		if id, err = utils.ReadUInt64(buf); err != nil {
			return fmt.Errorf("error read parent record ID: %w", err)
		}
		row.setParent(istructs.RecordID(id))
	}
	if (sysFieldMask & sfm_Container) == sfm_Container {
		var id uint16
		if id, err = utils.ReadUInt16(buf); err != nil {
			return fmt.Errorf("error read record container ID: %w", err)
		}
		if err = row.setContainerID(containers.ContainerID(id)); err != nil {
			return fmt.Errorf("error read record container: %w", err)
		}
	}
	if (sysFieldMask & sfm_IsActive) == sfm_IsActive {
		var a bool
		if a, err = utils.ReadBool(buf); err != nil {
			return fmt.Errorf("error read record is active: %w", err)
		}
		row.setActive(a)
	}
	return nil
}
