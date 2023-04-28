/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	dynobuffers "github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/containers"
	"github.com/voedger/voedger/pkg/istructsmem/internal/dynobuf"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

// rowType is type to implement istructs row interfaces.

//   - interfaces:
//     — istructs.IRowReader
//     — istructs.IRowWriter
//     — istructs.IValue
//     — istructs.IValueBuilder
//     — istructs.IRecord (partially)
//     — istructs.IEditableRecord
type rowType struct {
	appCfg    *AppConfigType
	schema    appdef.IDef
	id        istructs.RecordID
	parentID  istructs.RecordID
	container string
	isActive  bool
	dyB       *dynobuffers.Buffer
	err       error
}

// newRow constructs new row (QName is appdef.NullQName)
func newRow(appCfg *AppConfigType) rowType {
	return rowType{
		appCfg:    appCfg,
		schema:    appdef.NullDef,
		id:        istructs.NullRecordID,
		parentID:  istructs.NullRecordID,
		container: "",
		isActive:  true,
		dyB:       nullDynoBuffer,
		err:       nil,
	}
}

// build builds the row. Must be called after all Put××× calls to build row. If there were errors during data puts, then their connection will be returned.
// If there were no errors, then tries to form the dynobuffer and returns the result
func (row *rowType) build() (nilledFields []string, err error) {
	if row.err != nil {
		return nil, row.error()
	}

	if row.QName() == appdef.NullQName {
		return nil, nil
	}

	if row.dyB.IsModified() {
		var bytes []byte
		if bytes, nilledFields, err = row.dyB.ToBytesNilled(); err == nil {
			row.dyB.Reset(utils.CopyBytes(bytes))
			for _, nilledFieldName := range nilledFields {
				row.dyB.Set(nilledFieldName, nil)
			}
		}
	}

	return nilledFields, err
}

// clear clears row by set QName to NullQName value
func (row *rowType) clear() {
	row.schema = appdef.NullDef
	row.id = istructs.NullRecordID
	row.parentID = istructs.NullRecordID
	row.container = ""
	row.isActive = true
	row.dyB = nullDynoBuffer
	row.err = nil
}

// collectError collects errors that occur when puts data into a row
func (row *rowType) collectError(err error) {
	row.err = errors.Join(row.err, err)
}

func (row *rowType) collectErrorf(format string, a ...interface{}) {
	row.collectError(fmt.Errorf(format, a...))
}

// containerID returns row container id
func (row *rowType) containerID() (id containers.ContainerID, err error) {
	return row.appCfg.cNames.GetID(row.Container())
}

// copyFrom assigns from specified row
func (row *rowType) copyFrom(src *rowType) {
	row.clear()

	row.appCfg = src.appCfg
	row.setQName(src.QName())

	row.id = src.id
	row.parentID = src.parentID
	row.container = src.container
	row.isActive = src.isActive

	src.dyB.IterateFields(nil,
		func(name string, data interface{}) bool {
			row.dyB.Set(name, data)
			return true
		})

	_, _ = row.build()
}

// empty returns true if no data except system fields
func (row *rowType) empty() bool {
	userFields := false
	row.dyB.IterateFields(nil,
		func(name string, _ interface{}) bool {
			userFields = true
			return false
		})
	return !userFields
}

// error returns concatenation of collected errors. Errors are collected from Put××× methods fails
func (row *rowType) error() error {
	return row.err
}

// hasValue returns has dynobuffer data in specified field
func (row *rowType) hasValue(name string) (value bool) {
	if name == appdef.SystemField_QName {
		// special case: sys.QName is always presents
		return true
	}
	if name == appdef.SystemField_ID {
		return row.id != istructs.NullRecordID
	}
	if name == appdef.SystemField_ParentID {
		return row.parentID != istructs.NullRecordID
	}
	if name == appdef.SystemField_Container {
		return row.container != ""
	}
	if name == appdef.SystemField_IsActive {
		// special case: sys.IsActive is presents if schema required
		return row.schema.Kind().HasSystemField(appdef.SystemField_IsActive)
	}
	return row.dyB.HasValue(name)
}

// loadFromBytes loads row from bytes
func (row *rowType) loadFromBytes(in []byte) (err error) {

	buf := bytes.NewBuffer(in)

	var codec byte
	if err = binary.Read(buf, binary.BigEndian, &codec); err != nil {
		return fmt.Errorf("error read codec version: %w", err)
	}
	switch codec {
	case codec_RawDynoBuffer, codec_RDB_1:
		if err := loadRow(row, codec, buf); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown codec version «%d»: %w", codec, ErrUnknownCodec)
	}

	return nil
}

// maskValues masks values in row. Digital values are masked by zeros, strings — by star «*». System fields are not masked
func (row *rowType) maskValues() {
	row.dyB.IterateFields(nil,
		func(name string, data interface{}) bool {
			if _, ok := data.(string); ok {
				row.dyB.Set(name, maskString)
			} else {
				row.dyB.Set(name, nil)
			}
			return true
		})

	if row.dyB.IsModified() {
		bytes := row.dyB.GetBytes()
		row.dyB.Reset(utils.CopyBytes(bytes))
	}
}

// putValue checks is field specified name and kind exists in dynobuffers schema.
// If exists then puts specified field value into dynobuffer else collects error.
// Remark: if field must be verificated before put then collects error «field must be verified»
func (row *rowType) putValue(name string, kind dynobuffers.FieldType, value interface{}) {
	fld, ok := row.dyB.Scheme.FieldsMap[name]
	if !ok {
		row.collectErrorf(errFieldNotFoundWrap, dynobuf.FieldTypeToString(kind), name, row.QName(), ErrNameNotFound)
		return
	}

	if fld := row.schema.Field(name); fld != nil {
		if fld.Verifiable() {
			token, ok := value.(string)
			if !ok {
				row.collectErrorf(errFieldMustBeVerificated, name, value, ErrWrongFieldType)
				return
			}
			data, err := row.verifyToken(name, token)
			if err != nil {
				row.collectError(err)
				return
			}
			row.dyB.Set(name, data)
			return
		}
	}

	if (kind != dynobuffers.FieldTypeUnspecified) && (fld.Ft != kind) {
		row.collectErrorf(errFieldValueTypeMismatchWrap, dynobuf.FieldTypeToString(kind), dynobuf.FieldTypeToString(fld.Ft), name, ErrWrongFieldType)
		return
	}

	row.dyB.Set(name, value)
}

// qNameID returns storage ID of row QName
func (row *rowType) qNameID() (qnames.QNameID, error) {
	name := row.QName()
	if name == appdef.NullQName {
		return qnames.NullQNameID, nil
	}
	return row.appCfg.qNames.GetID(name)
}

// setActive sets record IsActive activity flag
func (row *rowType) setActive(value bool) {
	row.isActive = value
}

// setContainer sets record container
func (row *rowType) setContainer(value string) {
	if row.container != value {
		row.container = value
		if _, err := row.containerID(); err != nil {
			row.collectError(err)
		}
	}
}

// setContainerID sets record container by ID. Useful from loadFromBytes()
func (row *rowType) setContainerID(value containers.ContainerID) (err error) {
	cont, err := row.appCfg.cNames.GetContainer(value)
	if err != nil {
		row.collectError(err)
		return err
	}

	row.container = cont
	return nil
}

// setID sets record ID
func (row *rowType) setID(value istructs.RecordID) {
	row.id = value
}

// setParent sets record parent ID
func (row *rowType) setParent(value istructs.RecordID) {
	row.parentID = value
}

// setQName sets new specified QName for row. It resets all data from row
func (row *rowType) setQName(value appdef.QName) {
	if row.QName() == value {
		return
	}

	row.clear()

	if value == appdef.NullQName {
		return
	}

	schema := row.appCfg.AppDef.DefByName(value)
	if schema == nil {
		row.collectErrorf(errSchemaNotFoundWrap, value, ErrNameNotFound)
		return
	}

	row.setSchema(schema)
}

// setQNameID same as setQName, useful from loadFromBytes()
func (row *rowType) setQNameID(value qnames.QNameID) (err error) {
	if id, err := row.qNameID(); (err == nil) && (id == value) {
		return nil
	}

	row.clear()

	qName, err := row.appCfg.qNames.GetQName(value)
	if err != nil {
		row.collectError(err)
		return err
	}

	if qName != appdef.NullQName {
		schema := row.appCfg.AppDef.DefByName(qName)
		if schema == nil {
			err = fmt.Errorf(errSchemaNotFoundWrap, qName, ErrNameNotFound)
			row.collectError(err)
			return err
		}
		row.setSchema(schema)
	}

	return nil
}

// setSchema assign specified schema to row and rebuild row. Schema can not to be nil and must be valid
func (row *rowType) setSchema(value appdef.IDef) {
	if value == nil {
		row.schema = appdef.NullDef
	} else {
		row.schema = value
	}

	if row.schema.QName() == appdef.NullQName {
		row.dyB = nullDynoBuffer
	} else {
		row.dyB = dynobuffers.NewBuffer(row.appCfg.dynoSchemes[row.schema.QName()])
	}
}

// storeToBytes stores row to bytes and returns error if occurs
func (row *rowType) storeToBytes() (out []byte, err error) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, codec_LastVersion)

	if err := storeRow(row, buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// verifyToken verifies specified token for specified field and returns successfully verified token payload value or error
func (row *rowType) verifyToken(name string, token string) (value interface{}, err error) {
	payload := payloads.VerifiedValuePayload{}
	tokens := row.appCfg.app.AppTokens()
	if _, err = tokens.ValidateToken(token, &payload); err != nil {
		return nil, err
	}

	// if gpayload.AppQName != row.appCfg.Name { … } // redundant check, must be check by IAppToken.ValidateToken()
	// if expTime := gpayload.IssuedAt.Add(gpayload.Duration); time.Now().After(expTime) { … } // redundant check, must be check by IAppToken.ValidateToken()

	fld := row.schema.Field(name)

	if !fld.VerificationKind(payload.VerificationKind) {
		return nil, fmt.Errorf("unavailable verification method %v: %w", payload.VerificationKind, ErrInvalidVerificationKind)
	}

	if payload.Entity != row.QName() {
		return nil, fmt.Errorf("verified entity QName is «%v», but «%v» expected: %w", payload.Entity, row.QName(), ErrInvalidName)
	}
	if payload.Field != name {
		return nil, fmt.Errorf("verified field is «%s», but «%s» expected: %w", payload.Field, name, ErrInvalidName)
	}

	if value, err = row.dynoBufValue(payload.Value, fld.DataKind()); err != nil {
		return nil, fmt.Errorf("verified field «%s» data has invalid type: %w", name, err)
	}

	return value, nil
}

// istructs.IRowReader.AsInt32
func (row *rowType) AsInt32(name string) (value int32) {
	if value, ok := row.dyB.GetInt32(name); ok {
		return value
	}
	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_int32.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return 0
}

// istructs.IRowReader.AsInt64
func (row *rowType) AsInt64(name string) (value int64) {
	if value, ok := row.dyB.GetInt64(name); ok {
		return value
	}
	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_int64.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return 0
}

// istructs.IRowReader.AsFloat32
func (row *rowType) AsFloat32(name string) (value float32) {
	if value, ok := row.dyB.GetFloat32(name); ok {
		return value
	}
	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_float32.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return 0
}

// istructs.IRowReader.AsFloat64
func (row *rowType) AsFloat64(name string) (value float64) {
	if value, ok := row.dyB.GetFloat64(name); ok {
		return value
	}
	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_float64.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return 0
}

// istructs.IRowReader.AsBytes
func (row *rowType) AsBytes(name string) (value []byte) {
	if bytes := row.dyB.GetByteArray(name); bytes != nil {
		return bytes.Bytes()
	}
	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_bytes.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return nil
}

// istructs.IRowReader.AsString
func (row *rowType) AsString(name string) (value string) {
	if name == appdef.SystemField_Container {
		return row.container
	}

	if value, ok := row.dyB.GetString(name); ok {
		return value
	}

	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_string.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return ""
}

// istructs.IRowReader.AsQName
func (row *rowType) AsQName(name string) appdef.QName {
	if name == appdef.SystemField_QName {
		// special case: «sys.QName» field must returned from assigned schema
		return row.schema.QName()
	}

	if id, ok := dynoBufGetWord(row.dyB, name); ok {
		qName, err := row.appCfg.qNames.GetQName(qnames.QNameID(id))
		if err != nil {
			panic(err)
		}
		return qName
	}

	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_QName.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return appdef.NullQName
}

// istructs.IRowReader.AsBool
func (row *rowType) AsBool(name string) bool {
	if name == appdef.SystemField_IsActive {
		return row.isActive
	}

	if value, ok := row.dyB.GetBool(name); ok {
		return value
	}

	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_bool.ToString(), name, row.QName(), ErrNameNotFound))
	}

	return false
}

// istructs.IRowReader.AsRecordID
func (row *rowType) AsRecordID(name string) istructs.RecordID {
	if name == appdef.SystemField_ID {
		return row.id
	}

	if name == appdef.SystemField_ParentID {
		return row.parentID
	}

	if value, ok := row.dyB.GetInt64(name); ok {
		return istructs.RecordID(value)
	}

	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_RecordID.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return istructs.NullRecordID
}

// IValue.AsRecord
func (row *rowType) AsRecord(name string) istructs.IRecord {
	if bytes := row.dyB.GetByteArray(name); bytes != nil {
		record := newRecord(row.appCfg)
		if err := record.loadFromBytes(bytes.Bytes()); err != nil {
			panic(err)
		}
		return &record
	}
	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_Record.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return NewNullRecord(istructs.NullRecordID)
}

// IValue.AsEvent
func (row *rowType) AsEvent(name string) istructs.IDbEvent {
	if bytes := row.dyB.GetByteArray(name); bytes != nil {
		event := newDbEvent(row.appCfg)
		if err := event.loadFromBytes(bytes.Bytes()); err != nil {
			panic(err)
		}
		return &event
	}
	if row.schema.Field(name) == nil {
		panic(fmt.Errorf(errFieldNotFoundWrap, appdef.DataKind_Event.ToString(), name, row.QName(), ErrNameNotFound))
	}
	return nil
}

// istructs.IRecord.Container
func (row *rowType) Container() string {
	return row.container
}

// istructs.IRowReader.FieldNames
func (row *rowType) FieldNames(cb func(fieldName string)) {
	// system fields
	if row.schema.Kind().HasSystemField(appdef.SystemField_QName) {
		cb(appdef.SystemField_QName)
	}
	if row.id != istructs.NullRecordID {
		cb(appdef.SystemField_ID)
	}
	if row.parentID != istructs.NullRecordID {
		cb(appdef.SystemField_ParentID)
	}
	if row.container != "" {
		cb(appdef.SystemField_Container)
	}
	if row.schema.Kind().HasSystemField(appdef.SystemField_IsActive) {
		cb(appdef.SystemField_IsActive)
	}

	// user fields
	row.dyB.IterateFields(nil,
		func(name string, _ interface{}) bool {
			cb(name)
			return true
		})
}

// istructs.IRecord.ID
func (row *rowType) ID() istructs.RecordID {
	return row.id
}

// istructs.IEditableRecord.IsActive
func (row *rowType) IsActive() bool {
	return row.isActive
}

// istructs.IRecord.Parent
func (row *rowType) Parent() istructs.RecordID {
	return row.parentID
}

// istructs.IRowWriter.PutInt32
func (row *rowType) PutInt32(name string, value int32) {
	row.putValue(name, dynobuffers.FieldTypeInt32, value)
}

// istructs.IRowWriter.PutInt64
func (row *rowType) PutInt64(name string, value int64) {
	row.putValue(name, dynobuffers.FieldTypeInt64, value)
}

// istructs.IRowWriter.PutFloat32
func (row *rowType) PutFloat32(name string, value float32) {
	row.putValue(name, dynobuffers.FieldTypeFloat32, value)
}

// istructs.IRowWriter.PutFloat64
func (row *rowType) PutFloat64(name string, value float64) {
	row.putValue(name, dynobuffers.FieldTypeFloat64, value)
}

// istructs.IRowWriter.PutNumber
func (row *rowType) PutNumber(name string, value float64) {
	fld := row.schema.Field(name)
	if fld == nil {
		row.collectErrorf(errFieldNotFoundWrap, "number", name, row.QName(), ErrNameNotFound)
		return
	}

	switch k := fld.DataKind(); k {
	case appdef.DataKind_int32:
		row.dyB.Set(name, int32(value))
	case appdef.DataKind_int64:
		row.dyB.Set(name, int64(value))
	case appdef.DataKind_float32:
		row.dyB.Set(name, float32(value))
	case appdef.DataKind_float64:
		row.dyB.Set(name, value)
	case appdef.DataKind_RecordID:
		row.PutRecordID(name, istructs.RecordID(value))
	default:
		row.collectErrorf(errFieldValueTypeMismatchWrap, appdef.DataKind_float64.ToString(), k, name, ErrWrongFieldType)
	}
}

// istructs.IRowWriter.PutBytes
func (row *rowType) PutBytes(name string, value []byte) {
	row.putValue(name, dynobuffers.FieldTypeByte, value)
}

// istructs.IRowWriter.PutString
func (row *rowType) PutString(name string, value string) {
	if name == appdef.SystemField_Container {
		row.setContainer(value)
		return
	}
	row.putValue(name, dynobuffers.FieldTypeString, value)
}

// istructs.IRowWriter.PutQName
func (row *rowType) PutQName(name string, value appdef.QName) {
	if name == appdef.SystemField_QName {
		// special case: user try to assign empty record early constructed from CUD.Create()
		if row.QName() == appdef.NullQName {
			row.setQName(value)
		} else if row.QName() != value {
			row.collectErrorf("%w", ErrSchemaChanged)
		}
		return
	}

	id, err := row.appCfg.qNames.GetID(value)
	if err != nil {
		row.collectErrorf(errCantGetFieldQNameIDWrap, name, value, err)
		return
	}
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(id))

	row.putValue(name, dynobuffers.FieldTypeByte, b)
}

// istructs.IRowWriter.PutChars
func (row *rowType) PutChars(name string, value string) {
	fld := row.schema.Field(name)
	if fld == nil {
		row.collectErrorf(errFieldNotFoundWrap, "chars", name, row.QName(), ErrNameNotFound)
		return
	}

	switch k := fld.DataKind(); k {
	case appdef.DataKind_bytes:
		bytes, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			row.collectErrorf(errFieldConvertErrorWrap, name, value, appdef.DataKind_bytes.ToString(), err)
			return
		}
		row.PutBytes(name, bytes)
	case appdef.DataKind_string:
		row.PutString(name, value)
	case appdef.DataKind_QName:
		qName, err := appdef.ParseQName(value)
		if err != nil {
			row.collectErrorf(errFieldConvertErrorWrap, name, value, appdef.DataKind_QName.ToString(), err)
			return
		}
		row.PutQName(name, qName)
	default:
		row.collectErrorf(errFieldValueTypeMismatchWrap, appdef.DataKind_string.ToString(), k, name, ErrWrongFieldType)
	}
}

// istructs.IRowWriter.PutBool
func (row *rowType) PutBool(name string, value bool) {
	if name == appdef.SystemField_IsActive {
		row.setActive(value)
		return
	}

	row.putValue(name, dynobuffers.FieldTypeBool, value)
}

// istructs.IRowWriter.PutRecordID
func (row *rowType) PutRecordID(name string, value istructs.RecordID) {
	if name == appdef.SystemField_ID {
		row.setID(value)
		return
	}
	if name == appdef.SystemField_ParentID {
		row.setParent(value)
		return
	}

	row.putValue(name, dynobuffers.FieldTypeInt64, int64(value))
}

// istructs.IValueBuilder.PutRecord
func (row *rowType) PutRecord(name string, record istructs.IRecord) {
	if rec, ok := record.(*recordType); ok {
		if bytes, err := rec.storeToBytes(); err == nil {
			row.putValue(name, dynobuffers.FieldTypeByte, bytes)
		}
	}
}

// istructs.IValueBuilder.PutEvent
func (row *rowType) PutEvent(name string, event istructs.IDbEvent) {
	if ev, ok := event.(*dbEventType); ok {
		if bytes, err := ev.storeToBytes(); err == nil {
			row.putValue(name, dynobuffers.FieldTypeByte, bytes)
		}
	}
}

// istructs.IRecord.QName: returns row qualified name
func (row *rowType) QName() appdef.QName {
	if row.schema != nil {
		return row.schema.QName()
	}
	return appdef.NullQName
}

// istructs.IRowReader.RecordIDs
func (row *rowType) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {
	row.schema.Fields(
		func(fld appdef.Field) {
			if fld.DataKind() == appdef.DataKind_RecordID {
				id := row.AsRecordID(fld.Name())
				if (id != istructs.NullRecordID) || includeNulls {
					cb(fld.Name(), id)
				}
			}
		})
}
