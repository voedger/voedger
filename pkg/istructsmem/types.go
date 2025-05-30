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
	"errors"
	"fmt"

	"github.com/untillpro/dynobuffers"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/containers"
	"github.com/voedger/voedger/pkg/istructsmem/internal/dynobuf"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

// # Implements:
//   - istructs.IRowReader
//   - istructs.IRowWriter
//   - istructs.IValue
//   - istructs.IValueBuilder
//   - istructs.IRecord (partially)
//   - istructs.IEditableRecord
type rowType struct {
	appCfg           *AppConfigType
	typ              appdef.IType
	fields           appdef.IWithFields
	id               istructs.RecordID
	parentID         istructs.RecordID
	container        string
	isActive         bool
	isActiveModified bool
	dyB              *dynobuffers.Buffer
	nils             map[string]appdef.IField // emptied string- and []bytes- fields, which values are not stored in dynobuffer
	err              error
}

// Makes new empty row (QName is appdef.NullQName)
func makeRow(appCfg *AppConfigType) rowType {
	return rowType{
		appCfg:    appCfg,
		typ:       appdef.NullType,
		fields:    appdef.NullFields,
		id:        istructs.NullRecordID,
		parentID:  istructs.NullRecordID,
		container: "",
		isActive:  true,
		dyB:       nullDynoBuffer,
		nils:      nil,
		err:       nil,
	}
}

// makes new empty row (QName is appdef.NullQName)
func newRow(appCfg *AppConfigType) *rowType {
	r := makeRow(appCfg)
	return &r
}

// build builds the row. Must be called after all Put××× calls to build row. If there were errors during data puts, then their connection will be returned.
// If there were no errors, then tries to form the dynoBuffer and returns the result
func (row *rowType) build() error {
	if row.err != nil {
		return row.err
	}

	if row.QName() == appdef.NullQName {
		return nil
	}

	if row.dyB.IsModified() {
		bytes, err := row.dyB.ToBytes()
		if err != nil {
			return err
		}
		row.dyB.Reset(utils.CopyBytes(bytes))
	}

	return nil
}

// Checks is specified field is nullable (string- or []byte- type) and put value is nil or zero length.
// In this case adds field to nils map, otherwise removes field from nils map.
// #2785
func (row *rowType) checkPutNil(field appdef.IField, value any) {
	isNil := false

	switch field.DataKind() {
	case appdef.DataKind_string:
		if value == nil || len(value.(string)) == 0 {
			isNil = true
		}
	case appdef.DataKind_bytes:
		if value == nil || len(value.([]byte)) == 0 {
			isNil = true
		}
	default:
		return
	}

	if isNil {
		if row.nils == nil {
			row.nils = make(map[string]appdef.IField)
		}
		row.nils[field.Name()] = field
	} else if row.nils != nil {
		delete(row.nils, field.Name())
	}
}

// clear clears row by set QName to NullQName value
func (row *rowType) clear() {
	row.typ = appdef.NullType
	row.fields = appdef.NullFields
	row.id = istructs.NullRecordID
	row.parentID = istructs.NullRecordID
	row.container = ""
	row.isActive = true
	row.isActiveModified = false
	row.release()
	row.nils = nil
	row.err = nil
}

// collectError collects errors that occur when puts data into a row
func (row *rowType) collectError(err error) {
	row.err = errors.Join(row.err, err)
}

func (row *rowType) collectErrorf(format string, a ...any) {
	row.collectError(fmt.Errorf(format, a...))
}

// containerID returns row container id
func (row *rowType) containerID() (id containers.ContainerID, err error) {
	return row.appCfg.cNames.ID(row.Container())
}

// Assigns from specified row
func (row *rowType) copyFrom(src *rowType) {
	row.clear()

	row.appCfg = src.appCfg
	row.typ = src.typ
	row.fields = src.fields

	row.id = src.id
	row.parentID = src.parentID
	row.container = src.container
	row.isActive = src.isActive

	if src.dyB != nil {
		row.dyB = dynobuffers.NewBuffer(src.dyB.Scheme)
		src.dyB.IterateFields(nil,
			func(name string, data any) bool {
				row.dyB.Set(name, data)
				return true
			})
	}

	_ = row.build()
}

// Returns true if no data except system fields
func (row *rowType) empty() bool {
	userFields := false
	row.dyB.IterateFields(nil,
		func(name string, _ any) bool {
			userFields = true
			return false
		})
	return !userFields
}

// Returns specified field definition or nil if field not found
func (row *rowType) fieldDef(name appdef.FieldName) appdef.IField {
	return row.fields.Field(name)
}

// Returns specified typed field definition.
//
// # Panics:
//   - if field not found
//   - if field has different data kind
func (row *rowType) fieldMustExists(name appdef.FieldName, k appdef.DataKind, otherKinds ...appdef.DataKind) appdef.IField {
	f := row.fieldDef(name)
	if f != nil {
		if f.DataKind() == k {
			return f
		}
		for _, k := range otherKinds {
			if f.DataKind() == k {
				return f
			}
		}
	}
	panic(ErrTypedFieldNotFound(k.TrimString(), name, row))
}

// Loads row from bytes
func (row *rowType) loadFromBytes(in []byte) (err error) {

	buf := bytes.NewBuffer(in)

	var codec byte
	if codec, err = utils.ReadByte(buf); err != nil {
		return enrichError(err, "error read codec version")
	}
	switch codec {
	case codec_RawDynoBuffer, codec_RDB_1, codec_RDB_2:
		if err := loadRow(row, codec, buf); err != nil {
			return err
		}
	default:
		return ErrUnknownCodec(codec)
	}

	return nil
}

// Masks values in row. Digital values are masked by zeros, strings — by star «*». System fields are not masked
func (row *rowType) maskValues() {
	row.dyB.IterateFields(nil,
		func(name string, data any) bool {
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

// Checks is field specified name and kind exists in dynobuffers scheme.
//
// If exists then puts specified field value into dynoBuffer else collects error.
//
// If field has restricts (length, pattern, etc.) then checks value by field restricts.
//
// If field must be verified before put then collects error «field must be verified».
func (row *rowType) putValue(name appdef.FieldName, kind appdef.DataKind, value any) {

	if a, ok := row.typ.(appdef.IWithAbstract); ok {
		if a.Abstract() {
			row.collectError(ErrAbstractType("%v is abstract", row.QName()))
			return
		}
	}

	fld := row.fieldDef(name)
	if fld == nil {
		row.collectError(ErrFieldNotFound(name, row))
		return
	}

	switch name {
	case appdef.SystemField_ID:
		if int64Val, ok := value.(int64); ok {
			row.setID(istructs.RecordID(int64Val)) // nolint G115
		} else {
			row.collectError(ErrWrongType("%T not applicable for %v", value, fld))
		}
		return
	case appdef.SystemField_ParentID:
		if int64Val, ok := value.(int64); ok {
			row.setParent(istructs.RecordID(int64Val)) // nolint G115
		} else {
			row.collectError(ErrWrongType("%T not applicable for %v", value, fld))
		}
		return
	}

	fieldValue := value

	if fld.Verifiable() {
		if token, ok := value.(string); ok {
			if data, err := row.verifyToken(fld, token); err == nil {
				fieldValue = data // override value with verified value
			} else {
				row.collectError(err)
				return
			}
		} else {
			row.collectError(ErrWrongFieldType("%v should be verified, expected token, but value «%T» passed", fld, value))
			return
		}
	} else {
		if f, ok := row.dyB.Scheme.FieldsMap[name]; ok {
			if k := dynobuf.DataKindToFieldType(kind); f.Ft != k {
				row.collectError(ErrWrongFieldType("can not put %s to %v", kind.TrimString(), fld))
				return
			}
		}
	}

	if err := checkConstraints(fld, fieldValue); err != nil {
		row.collectError(err)
		return
	}

	row.checkPutNil(fld, fieldValue) // #2785

	switch fld.DataKind() {
	case appdef.DataKind_int8: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
		row.dyB.Set(name, byte(fieldValue.(int8))) // nolint G115 : dynobuffers uses byte to store int8
	default:
		row.dyB.Set(name, fieldValue)
	}
}

// QNameID returns storage ID of row QName
func (row *rowType) QNameID() (istructs.QNameID, error) {
	name := row.QName()
	if name == appdef.NullQName {
		return istructs.NullQNameID, nil
	}
	return row.appCfg.qNames.ID(name)
}

// Returns dynobuffer to pull
func (row *rowType) release() {
	if row.dyB != nullDynoBuffer {
		row.dyB.Release()
		row.dyB = nullDynoBuffer
	}
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
	cont, err := row.appCfg.cNames.Container(value)
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

	t := row.appCfg.AppDef.Type(value)
	if t == appdef.NullType {
		row.collectError(ErrTypeNotFound(value))
		return
	}

	row.setType(t)
}

// Same as setQName, useful from loadFromBytes()
func (row *rowType) setQNameID(value istructs.QNameID) (err error) {
	if id, err := row.QNameID(); (err == nil) && (id == value) {
		return nil
	}

	row.clear()

	qName, err := row.appCfg.qNames.QName(value)
	if err != nil {
		row.collectError(err)
		return err
	}

	if qName != appdef.NullQName {
		t := row.appCfg.AppDef.Type(qName)
		if t == appdef.NullType {
			err = ErrTypeNotFound(qName)
			row.collectError(err)
			return err
		}
		row.setType(t)
	}

	return nil
}

// Assign specified type to row and rebuild row.
//
// Type can be nil, this will clear row.
// If type is not nil, then type may be:
//   - any structured type (doc or record),
//   - view value
func (row *rowType) setType(t appdef.IType) {
	row.clear()

	if t == nil {
		row.typ = appdef.NullType
		row.fields = appdef.NullFields
	} else {
		row.typ = t
		if v, ok := t.(appdef.IView); ok {
			row.fields = v.Value()
			row.dyB = dynobuffers.NewBuffer(row.appCfg.dynoSchemes.Scheme(t.QName()))
		} else {
			if f, ok := t.(appdef.IWithFields); ok {
				row.fields = f
				row.dyB = dynobuffers.NewBuffer(row.appCfg.dynoSchemes.Scheme(t.QName()))
			} else {
				// notest
				row.collectError(ErrWrongType("%v has no fields", t))
			}
		}
	}
}

// Assign specified view partition key to row and rebuild row.
//
// View can be nil, this will clear row.
func (row *rowType) setViewPartKey(v appdef.IView) {
	row.clear()

	if v != nil {
		row.typ = v
		row.fields = v.Key().PartKey()
		row.dyB = dynobuffers.NewBuffer(row.appCfg.dynoSchemes.ViewPartKeyScheme(v.QName()))
	}
}

// Assign specified view clustering columns to row and rebuild row.
//
// View can be nil, this will clear row.
func (row *rowType) setViewClustCols(v appdef.IView) {
	row.clear()

	if v != nil {
		row.typ = v
		row.fields = v.Key().ClustCols()
		row.dyB = dynobuffers.NewBuffer(row.appCfg.dynoSchemes.ViewClustColsScheme(v.QName()))
	}
}

// Stores row to bytes.
//
// # Panics:
//
//   - Must be called *after* event validation. Overwise function may panic!
func (row *rowType) storeToBytes() []byte {
	buf := new(bytes.Buffer)
	utils.WriteByte(buf, codec_LastVersion)

	storeRow(row, buf)

	return buf.Bytes()
}

// returns row type definition
func (row *rowType) typeDef() appdef.IType {
	return row.typ
}

// verifyToken verifies specified token for specified field and returns successfully verified token payload value or error
func (row *rowType) verifyToken(fld appdef.IField, token string) (value any, err error) {
	payload := payloads.VerifiedValuePayload{}
	tokens := row.appCfg.app.AppTokens()
	if _, err = tokens.ValidateToken(token, &payload); err != nil {
		return nil, err
	}

	// if payload.AppQName != row.appCfg.Name { … } // redundant check, must be check by IAppToken.ValidateToken()
	// if expTime := payload.IssuedAt.Add(payload.Duration); time.Now().After(expTime) { … } // redundant check, must be check by IAppToken.ValidateToken()

	if !fld.VerificationKind(payload.VerificationKind) {
		return nil, ErrInvalidVerificationKind(row, fld, payload.VerificationKind)
	}

	if payload.Entity != row.QName() {
		return nil, ErrInvalidName("verified entity is «%v», but «%v» expected", payload.Entity, row.QName())
	}
	if payload.Field != fld.Name() {
		return nil, ErrInvalidName("%v verified field is «%s», but «%s» expected", row, payload.Field, fld.Name())
	}

	if value, err = row.clarifyJSONValue(payload.Value, fld.DataKind()); err != nil {
		return nil, enrichError(err, "wrong value for %v verified %v", row, fld)
	}

	return value, nil
}

// #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
//
// istructs.IRowReader.AsInt8
func (row *rowType) AsInt8(name appdef.FieldName) int8 {
	_ = row.fieldMustExists(name, appdef.DataKind_int8)
	if value, ok := row.dyB.GetByte(name); ok {
		return int8(value) // nolint G115 : dynobuffers uses byte to store int8
	}
	return 0
}

// #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
//
// istructs.IRowReader.AsInt16
func (row *rowType) AsInt16(name appdef.FieldName) int16 {
	_ = row.fieldMustExists(name, appdef.DataKind_int16)
	if value, ok := row.dyB.GetInt16(name); ok {
		return value
	}
	return 0
}

// istructs.IRowReader.AsInt32
func (row *rowType) AsInt32(name appdef.FieldName) (value int32) {
	_ = row.fieldMustExists(name, appdef.DataKind_int32)
	if value, ok := row.dyB.GetInt32(name); ok {
		return value
	}
	return 0
}

// istructs.IRowReader.AsInt64
func (row *rowType) AsInt64(name appdef.FieldName) (value int64) {
	fld := row.fieldMustExists(name, appdef.DataKind_int64, appdef.DataKind_RecordID)

	if fld.DataKind() == appdef.DataKind_RecordID {
		switch name {
		case appdef.SystemField_ID:
			return int64(row.id) // nolint G115 TODO: data loss on sending RecordID to the client as a func response
		case appdef.SystemField_ParentID:
			return int64(row.parentID) // nolint G115 TODO: data loss on sending RecordID to the client as a func response
		}
	}

	if value, ok := row.dyB.GetInt64(name); ok {
		return value
	}
	return 0
}

// istructs.IRowReader.AsFloat32
func (row *rowType) AsFloat32(name appdef.FieldName) (value float32) {
	_ = row.fieldMustExists(name, appdef.DataKind_float32)
	if value, ok := row.dyB.GetFloat32(name); ok {
		return value
	}
	return 0
}

// istructs.IRowReader.AsFloat64
func (row *rowType) AsFloat64(name appdef.FieldName) (value float64) {
	fld := row.fieldMustExists(name, appdef.DataKind_float64,
		appdef.DataKind_int8, appdef.DataKind_int16, // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
		appdef.DataKind_int32, appdef.DataKind_int64, appdef.DataKind_float32, appdef.DataKind_RecordID)
	switch fld.DataKind() {
	case appdef.DataKind_int8: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
		if value, ok := row.dyB.GetByte(name); ok {
			return float64(value)
		}
	case appdef.DataKind_int16: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
		if value, ok := row.dyB.GetInt16(name); ok {
			return float64(value)
		}
	case appdef.DataKind_int32:
		if value, ok := row.dyB.GetInt32(name); ok {
			return float64(value)
		}
	case appdef.DataKind_int64:
		if value, ok := row.dyB.GetInt64(name); ok {
			return float64(value)
		}
	case appdef.DataKind_RecordID:
		switch name {
		case appdef.SystemField_ID:
			return float64(row.id)
		case appdef.SystemField_ParentID:
			return float64(row.parentID)
		}
		if value, ok := row.dyB.GetInt64(name); ok {
			return float64(value)
		}
	case appdef.DataKind_float32:
		if value, ok := row.dyB.GetFloat32(name); ok {
			return float64(value)
		}
	case appdef.DataKind_float64:
		if value, ok := row.dyB.GetFloat64(name); ok {
			return value
		}
	}
	return 0
}

// istructs.IRowReader.AsBytes
func (row *rowType) AsBytes(name appdef.FieldName) (value []byte) {
	_ = row.fieldMustExists(name, appdef.DataKind_bytes)
	if bytes := row.dyB.GetByteArray(name); bytes != nil {
		return bytes.Bytes()
	}

	return nil
}

// istructs.IRowReader.AsString
func (row *rowType) AsString(name appdef.FieldName) (value string) {
	if name == appdef.SystemField_Container {
		return row.container
	}

	_ = row.fieldMustExists(name, appdef.DataKind_string)

	if value, ok := row.dyB.GetString(name); ok {
		return value
	}

	return ""
}

// istructs.IRowReader.AsQName
func (row *rowType) AsQName(name appdef.FieldName) appdef.QName {
	if name == appdef.SystemField_QName {
		// special case: «sys.QName» field must returned from row type
		return row.typ.QName()
	}

	_ = row.fieldMustExists(name, appdef.DataKind_QName)

	if id, ok := dynoBufGetWord(row.dyB, name); ok {
		qName, err := row.appCfg.qNames.QName(id)
		if err != nil {
			panic(err)
		}
		return qName
	}

	return appdef.NullQName
}

// istructs.IRowReader.AsBool
func (row *rowType) AsBool(name appdef.FieldName) bool {
	if name == appdef.SystemField_IsActive {
		return row.isActive
	}

	_ = row.fieldMustExists(name, appdef.DataKind_bool)

	if value, ok := row.dyB.GetBool(name); ok {
		return value
	}

	return false
}

// istructs.IRowReader.AsRecordID
func (row *rowType) AsRecordID(name appdef.FieldName) istructs.RecordID {
	if name == appdef.SystemField_ID {
		return row.id
	}

	if name == appdef.SystemField_ParentID {
		return row.parentID
	}

	_ = row.fieldMustExists(name, appdef.DataKind_RecordID, appdef.DataKind_int64)

	if value, ok := row.dyB.GetInt64(name); ok {
		return istructs.RecordID(value) // nolint G115
	}

	return istructs.NullRecordID
}

// IValue.AsRecord
func (row *rowType) AsRecord(name appdef.FieldName) istructs.IRecord {
	_ = row.fieldMustExists(name, appdef.DataKind_Record)

	if bytes := row.dyB.GetByteArray(name); bytes != nil {
		rec := newRecord(row.appCfg)
		if err := rec.loadFromBytes(bytes.Bytes()); err != nil {
			panic(err)
		}
		return rec
	}

	return NewNullRecord(istructs.NullRecordID)
}

// IValue.AsEvent
func (row *rowType) AsEvent(name appdef.FieldName) istructs.IDbEvent {
	_ = row.fieldMustExists(name, appdef.DataKind_Event)

	if bytes := row.dyB.GetByteArray(name); bytes != nil {
		event := newEvent(row.appCfg)
		if err := event.loadFromBytes(bytes.Bytes()); err != nil {
			panic(err)
		}
		return event
	}

	return nil
}

// istructs.IRecord.Container
func (row *rowType) Container() string {
	return row.container
}

// istructs.IRowReader.Fields
func (row *rowType) Fields(cb func(appdef.IField) bool) {
	qNameField := row.fieldDef(appdef.SystemField_QName)
	if qNameField != nil {
		if !cb(qNameField) {
			return
		}
	}
	if row.id != istructs.NullRecordID {
		if !cb(row.fieldDef(appdef.SystemField_ID)) {
			return
		}
	}
	if row.parentID != istructs.NullRecordID {
		if !cb(row.fieldDef(appdef.SystemField_ParentID)) {
			return
		}
	}
	if row.container != "" {
		if !cb(row.fieldDef(appdef.SystemField_Container)) {
			return
		}
	}
	if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_IsActive); exists {
		if !cb(row.fieldDef(appdef.SystemField_IsActive)) {
			return
		}
	}

	// user fields
	row.dyB.IterateFields(nil,
		func(name string, _ any) bool {
			return cb(row.fieldDef(name))
		})
}

// FIXME: remove when no longer in use
//
// Returns has dynoBuffer data in specified field
func (row *rowType) HasValue(name appdef.FieldName) (value bool) {
	if name == appdef.SystemField_QName {
		// special case: sys.QName is always presents
		return row.typ.QName() != appdef.NullQName
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
		// special case: sys.IsActive is presents if exists for type kind
		exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_IsActive)
		return exists
	}
	return row.dyB.HasValue(name)
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

// #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
//
// istructs.IRowWriter.PutInt8
func (row *rowType) PutInt8(name appdef.FieldName, value int8) {
	row.putValue(name, appdef.DataKind_int8, value)
}

// #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
//
// istructs.IRowWriter.PutInt16
func (row *rowType) PutInt16(name appdef.FieldName, value int16) {
	row.putValue(name, appdef.DataKind_int16, value)
}

// istructs.IRowWriter.PutInt32
func (row *rowType) PutInt32(name appdef.FieldName, value int32) {
	row.putValue(name, appdef.DataKind_int32, value)
}

// istructs.IRowWriter.PutInt64
func (row *rowType) PutInt64(name appdef.FieldName, value int64) {
	row.putValue(name, appdef.DataKind_int64, value)
}

// istructs.IRowWriter.PutFloat32
func (row *rowType) PutFloat32(name appdef.FieldName, value float32) {
	row.putValue(name, appdef.DataKind_float32, value)
}

// istructs.IRowWriter.PutFloat64
func (row *rowType) PutFloat64(name appdef.FieldName, value float64) {
	row.putValue(name, appdef.DataKind_float64, value)
}

// istructs.IRowWriter.PutFromJSON
func (row *rowType) PutFromJSON(j map[appdef.FieldName]any) {
	if v, ok := j[appdef.SystemField_QName]; ok {
		switch qNameFieldValue := v.(type) {
		case string:
			qName, err := appdef.ParseQName(qNameFieldValue)
			if err != nil {
				row.collectError(enrichError(err, "can not parse value for field «%s»", appdef.SystemField_QName))
				return
			}
			row.setQName(qName)
		case appdef.QName:
			row.setQName(qNameFieldValue)
		default:
			row.collectError(ErrWrongFieldType("can not put «%T» to field «%s»", v, appdef.SystemField_QName))
			return
		}
	}

	if (row.QName() == appdef.NullQName) && (len(j) > 0) {
		row.collectError(ErrFieldIsEmpty(appdef.SystemField_QName))
		return
	}

	for n, v := range j {
		switch fv := v.(type) {
		case float64:
			row.PutFloat64(n, fv)
		case int8: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
			row.PutInt8(n, fv)
		case int16: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
			row.PutInt16(n, fv)
		case int32:
			row.PutInt32(n, fv)
		case int64:
			row.PutInt64(n, fv)
		case float32:
			row.PutFloat32(n, fv)
		case json.Number:
			row.PutNumber(n, fv)
		case istructs.RecordID:
			row.PutRecordID(n, fv)
		case string:
			row.PutChars(n, fv)
		case bool:
			row.PutBool(n, fv)
		case []byte:
			// happens e.g. on IRowWriter.PutJSON() after read from the storage
			row.PutBytes(n, fv)
		case appdef.QName:
			// happens if `j` is got from coreutils.FieldsToMap()
			if n != appdef.SystemField_QName {
				row.PutQName(n, fv)
			}
		default:
			row.collectError(ErrWrongType(`%#T for field "%s" with value %v`, v, n, v))
		}
	}
}

// istructs.IRowWriter.PutNumber
func (row *rowType) PutNumber(name appdef.FieldName, value json.Number) {
	fld := row.fieldDef(name)
	if fld == nil {
		row.collectError(ErrFieldNotFound(name, row))
		return
	}
	clarifiedVal, err := row.clarifyJSONValue(value, fld.DataKind())
	if err != nil {
		row.collectError(enrichError(err, "can not put %T to %v", value, fld))
		return
	}
	switch fld.DataKind() {
	case appdef.DataKind_int8: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
		row.PutInt8(name, clarifiedVal.(int8))
	case appdef.DataKind_int16: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
		row.PutInt16(name, clarifiedVal.(int16))
	case appdef.DataKind_int32:
		row.PutInt32(name, clarifiedVal.(int32))
	case appdef.DataKind_int64:
		row.PutInt64(name, clarifiedVal.(int64))
	case appdef.DataKind_float32:
		row.PutFloat32(name, clarifiedVal.(float32))
	case appdef.DataKind_float64:
		row.PutFloat64(name, clarifiedVal.(float64))
	case appdef.DataKind_RecordID:
		row.PutRecordID(name, clarifiedVal.(istructs.RecordID))
	default:
		// notest: avoided already by row.clarifyJSONValue()
		panic(ErrWrongFieldType("can not put json.Number to %v", fld))
	}
}

// istructs.IRowWriter.PutBytes
func (row *rowType) PutBytes(name appdef.FieldName, value []byte) {
	row.putValue(name, appdef.DataKind_bytes, value)
}

// istructs.IRowWriter.PutString
func (row *rowType) PutString(name appdef.FieldName, value string) {
	if name == appdef.SystemField_Container {
		row.setContainer(value)
		return
	}
	row.putValue(name, appdef.DataKind_string, value)
}

// istructs.IRowWriter.PutQName
func (row *rowType) PutQName(name appdef.FieldName, value appdef.QName) {
	if name == appdef.SystemField_QName {
		// special case: user try to assign empty record early constructed from CUD.Create()
		if row.QName() == appdef.NullQName {
			row.setQName(value)
		} else if row.QName() != value {
			row.collectError(ErrUnableToUpdateSystemField(row, appdef.SystemField_QName))
		}
		return
	}

	id, err := row.appCfg.qNames.ID(value)
	if err != nil {
		row.collectError(enrichError(err, "can not get ID for field «%s»", name))
		return
	}
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, id)

	row.putValue(name, appdef.DataKind_QName, b)
}

// istructs.IRowWriter.PutChars
func (row *rowType) PutChars(name appdef.FieldName, value string) {
	fld := row.fieldDef(name)
	if fld == nil {
		row.collectError(ErrFieldNotFound(name, row))
		return
	}

	switch k := fld.DataKind(); k {
	case appdef.DataKind_bytes:
		bytes, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			row.collectError(enrichError(err, "can not decode value for %v", fld))
			return
		}
		row.PutBytes(name, bytes)
	case appdef.DataKind_string:
		row.PutString(name, value)
	case appdef.DataKind_QName:
		qName, err := appdef.ParseQName(value)
		if err != nil {
			row.collectError(enrichError(err, "can not parse value for %v", fld))
			return
		}
		row.PutQName(name, qName)
	default:
		row.collectError(ErrWrongFieldType("can not put string to %v", fld))
	}
}

// istructs.IRowWriter.PutBool
func (row *rowType) PutBool(name appdef.FieldName, value bool) {
	if name == appdef.SystemField_IsActive {
		row.setActive(value)
		row.isActiveModified = true
		return
	}

	row.putValue(name, appdef.DataKind_bool, value)
}

// istructs.IRowWriter.PutRecordID
func (row *rowType) PutRecordID(name appdef.FieldName, value istructs.RecordID) {
	row.putValue(name, appdef.DataKind_RecordID, int64(value)) // nolint G115
}

// istructs.IValueBuilder.PutRecord
func (row *rowType) PutRecord(name appdef.FieldName, record istructs.IRecord) {
	if rec, ok := record.(*recordType); ok {
		bytes := rec.storeToBytes()
		row.putValue(name, appdef.DataKind_Record, bytes)
	}
}

// istructs.IValueBuilder.PutEvent
func (row *rowType) PutEvent(name appdef.FieldName, event istructs.IDbEvent) {
	if ev, ok := event.(*eventType); ok {
		bytes := ev.storeToBytes()
		row.putValue(name, appdef.DataKind_Event, bytes)
	}
}

// istructs.IRecord.QName: returns row qualified name
func (row *rowType) QName() appdef.QName {
	return row.typ.QName()
}

// istructs.IRowReader.RecordIDs
func (row *rowType) RecordIDs(includeNulls bool) func(cb func(appdef.FieldName, istructs.RecordID) bool) {
	return func(cb func(appdef.FieldName, istructs.RecordID) bool) {
		for _, fld := range row.fields.Fields() {
			if fld.DataKind() == appdef.DataKind_RecordID {
				id := row.AsRecordID(fld.Name())
				if (id != istructs.NullRecordID) || includeNulls {
					if !cb(fld.Name(), id) {
						break
					}
				}
			}
		}
	}
}

// Return readable name of row.
//
// If row has no QName (NullQName) then returns "null row".
//
// If row has container name, then the result complete like `CRecord «Price: sales.PriceRecord»`.
// Otherwise it will be short form, such as "CDoc «sales.BillDocument»".
func (row rowType) String() string {
	qName := row.AsQName(appdef.SystemField_QName)
	if qName == appdef.NullQName {
		return "null row"
	}

	kind := row.typ.Kind().TrimString()

	if n := row.Container(); n != "" {
		// complete form, such as "CRecord «Price: sales.PriceRecord»"
		return fmt.Sprintf("%s «%s: %s»", kind, n, qName.String())
	}

	// short form, such as "CDoc «sales.BillDocument»"
	return fmt.Sprint(row.typ)
}

type BuiltinJob struct {
	Name appdef.QName
	Func func(state istructs.IState, intents istructs.IIntents) error
}
