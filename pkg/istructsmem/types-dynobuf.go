/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	dynobuffers "github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

// dynoBufValue converts specified value to dynobuffer compatable type using specified data kind.
// If value type is not corresponding to kind then next conversions are available:
//
//	— float64 value can be converted to all numeric kinds (int32, int64, float32, float64, RecordID)
//	— string value can be converted to QName and []byte kinds
//
// QName values, record- and event- values returned as []byte
func (row *rowType) dynoBufValue(value interface{}, kind istructs.DataKindType) (interface{}, error) {
	switch kind {
	case istructs.DataKind_int32:
		switch v := value.(type) {
		case int32:
			return v, nil
		case float64:
			return int32(v), nil
		}
	case istructs.DataKind_int64:
		switch v := value.(type) {
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		}
	case istructs.DataKind_float32:
		switch v := value.(type) {
		case float32:
			return v, nil
		case float64:
			return float32(v), nil
		}
	case istructs.DataKind_float64:
		switch v := value.(type) {
		case float64:
			return v, nil
		}
	case istructs.DataKind_bytes:
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
	case istructs.DataKind_string:
		switch v := value.(type) {
		case string:
			return v, nil
		}
	case istructs.DataKind_QName:
		switch v := value.(type) {
		case string:
			qName, err := istructs.ParseQName(v)
			if err != nil {
				return nil, err
			}
			id, err := row.appCfg.qNames.qNameToID(qName)
			if err != nil {
				return nil, err
			}
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, uint16(id))
			return b, nil
		case istructs.QName:
			id, err := row.appCfg.qNames.qNameToID(v)
			if err != nil {
				return nil, err
			}
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, uint16(id))
			return b, nil
		}
	case istructs.DataKind_bool:
		switch v := value.(type) {
		case bool:
			return v, nil
		}
	case istructs.DataKind_RecordID:
		switch v := value.(type) {
		case float64:
			return int64(v), nil
		case istructs.RecordID:
			return int64(v), nil
		}
	case istructs.DataKind_Record:
		switch v := value.(type) {
		case *recordType:
			bytes, err := v.storeToBytes()
			if err != nil {
				return nil, err
			}
			return bytes, nil
		}
	case istructs.DataKind_Event:
		switch v := value.(type) {
		case *dbEventType:
			bytes, err := v.storeToBytes()
			if err != nil {
				return nil, err
			}
			return bytes, nil
		}
	}
	return nil, fmt.Errorf("value has type «%T», but «%s» expected: %w", value, dataKindToStr[kind], ErrWrongFieldType)
}

func dynoBufGetWord(dyB *dynobuffers.Buffer, fieldName string) (value uint16, ok bool) {
	if b := dyB.GetByteArray(fieldName); b != nil {
		if bytes := b.Bytes(); len(bytes) == 2 {
			value = binary.BigEndian.Uint16(bytes)
			return value, true
		}
	}
	return 0, false
}

func storeRow(row *rowType, buf *bytes.Buffer) (err error) {
	id, err := row.qNameID()
	if err != nil {
		return err
	}
	_ = binary.Write(buf, binary.BigEndian, int16(id))
	if row.QName() == istructs.NullQName {
		return nil
	}

	if err = storeRowSysFields(row, buf); err != nil {
		return err
	}

	b, err := row.dyB.ToBytes()
	if err != nil {
		return err
	}
	len := uint32(len(b))
	_ = binary.Write(buf, binary.BigEndian, &len)
	_, _ = buf.Write(b)

	return nil
}

func storeRowSysFields(row *rowType, buf *bytes.Buffer) (err error) {
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

	_ = binary.Write(buf, binary.BigEndian, sysFieldMask)

	if row.ID() != istructs.NullRecordID {
		_ = binary.Write(buf, binary.BigEndian, uint64(row.ID()))
	}
	if row.parentID != istructs.NullRecordID {
		_ = binary.Write(buf, binary.BigEndian, uint64(row.parentID))
	}
	if row.container != "" {
		id, err := row.containerID()
		if err != nil {
			return err
		}
		_ = binary.Write(buf, binary.BigEndian, int16(id))
	}
	if !row.isActive {
		_ = binary.Write(buf, binary.BigEndian, false)
	}

	return nil
}

func loadRow(row *rowType, codecVer byte, buf *bytes.Buffer) (err error) {
	row.clear()

	var qnameId uint16
	if err = binary.Read(buf, binary.BigEndian, &qnameId); err != nil {
		return fmt.Errorf("error read row QNameID: %w", err)
	}
	if err = row.setQNameID(QNameID(qnameId)); err != nil {
		return err
	}
	if row.QName() == istructs.NullQName {
		return nil
	}

	if err = loadRowSysFields(row, codecVer, buf); err != nil {
		return err
	}

	len := uint32(0)
	if err := binary.Read(buf, binary.BigEndian, &len); err != nil {
		return fmt.Errorf("error read dynobuffer length: %w", err)
	}
	if buf.Len() < int(len) {
		return fmt.Errorf("error read dynobuffer, expected %d bytes, but only %d bytes is available: %w", len, buf.Len(), io.ErrUnexpectedEOF)
	}
	row.dyB.Reset(buf.Next(int(len)))

	return nil
}

// Returns system fields mask combination for schema kind, see sfm_××× consts
func schemaSysFieldsMask(schema *schemas.Schema) uint16 {
	sfm := uint16(0)
	if schema.Props().HasSystemField(istructs.SystemField_ID) {
		sfm |= sfm_ID
	}
	if schema.Props().HasSystemField(istructs.SystemField_ParentID) {
		sfm |= sfm_ParentID
	}
	if schema.Props().HasSystemField(istructs.SystemField_Container) {
		sfm |= sfm_Container
	}
	if schema.Props().HasSystemField(istructs.SystemField_IsActive) {
		sfm |= sfm_IsActive
	}
	return sfm
}

func loadRowSysFields(row *rowType, codecVer byte, buf *bytes.Buffer) (err error) {
	var sysFieldMask uint16

	if codecVer == codec_RawDynoBuffer {
		sysFieldMask = schemaSysFieldsMask(row.schema)
	} else {
		if err = binary.Read(buf, binary.BigEndian, &sysFieldMask); err != nil {
			return fmt.Errorf("error read system fields mask: %w", err)
		}
	}

	if (sysFieldMask & sfm_ID) == sfm_ID {
		var id uint64
		if err = binary.Read(buf, binary.BigEndian, &id); err != nil {
			return fmt.Errorf("error read record ID: %w", err)
		}
		row.setID(istructs.RecordID(id))
	}
	if (sysFieldMask & sfm_ParentID) == sfm_ParentID {
		var id uint64
		if err = binary.Read(buf, binary.BigEndian, &id); err != nil {
			return fmt.Errorf("error read parent record ID: %w", err)
		}
		row.setParent(istructs.RecordID(id))
	}
	if (sysFieldMask & sfm_Container) == sfm_Container {
		var id uint16
		if err = binary.Read(buf, binary.BigEndian, &id); err != nil {
			return fmt.Errorf("error read record container ID: %w", err)
		}
		if err = row.setContainerID(containerNameIDType(id)); err != nil {
			return fmt.Errorf("error read record container: %w", err)
		}
	}
	if (sysFieldMask & sfm_IsActive) == sfm_IsActive {
		var a bool
		if err = binary.Read(buf, binary.BigEndian, &a); err != nil {
			return fmt.Errorf("error read record is active: %w", err)
		}
		row.setActive(a)
	}
	return nil
}
