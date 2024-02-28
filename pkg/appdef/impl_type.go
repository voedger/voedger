/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// # Implements:
//   - IType
//   - ITypeBuilder
type typ struct {
	comment
	app  *appDef
	name QName
	kind TypeKind
}

// Creates and returns new type.
//
// Name can be empty (NullQName), then type is anonymous.
func makeType(app *appDef, name QName, kind TypeKind) typ {
	if name != NullQName {
		if ok, err := ValidQName(name); !ok {
			panic(fmt.Errorf("invalid type name «%v»: %w", name, err))
		}
	}
	return typ{comment{}, app, name, kind}
}

func (t *typ) App() IAppDef {
	return t.app
}

func (t *typ) IsSystem() bool {
	return t.QName().Pkg() == SysPackage
}

func (t *typ) Kind() TypeKind {
	return t.kind
}

func (t *typ) QName() QName {
	return t.name
}

func (t *typ) String() string {
	return fmt.Sprintf("%s «%v»", t.Kind().TrimString(), t.QName())
}

type typeRef struct {
	name QName
	t    IType
}

// Returns type by reference.
//
// If type is not found then returns nil.
func (r *typeRef) target(tt IWithTypes) IType {
	if r.name == NullQName {
		return nil
	}
	if r.name == QNameANY {
		return AnyType
	}
	if (r.t == nil) || (r.t.QName() != r.name) {
		r.t = tt.TypeByName(r.name)
	}
	return r.t
}

// Sets reference name
func (r *typeRef) setName(n QName) {
	r.name = n
	r.t = nil
}

// Returns is reference valid
func (r *typeRef) valid(tt IWithTypes) (bool, error) {
	if (r.name == NullQName) || (r.name == QNameANY) || (r.target(tt) != nil) {
		return true, nil
	}
	return false, fmt.Errorf("type «%v» is not found: %w", r.name, ErrNameNotFound)
}

// Validate specified type.
//
// # Validation:
//   - if type supports Validate() interface, then call this,
//   - if structured type has fields, validate fields,
//   - if structured type has containers, validate containers
func validateType(t IType) (err error) {
	if v, ok := t.(interface{ Validate() error }); ok {
		err = v.Validate()
	}

	if _, ok := t.(IFields); ok {
		err = errors.Join(err, validateTypeFields(t))
	}

	if _, ok := t.(IContainers); ok {
		err = errors.Join(err, validateTypeContainers(t))
	}

	return err
}

const nullTypeString = "null type"

type nullType struct{ nullComment }

func (t *nullType) App() IAppDef   { return nil }
func (t *nullType) IsSystem() bool { return false }
func (t *nullType) Kind() TypeKind { return TypeKind_null }
func (t *nullType) QName() QName   { return NullQName }
func (t *nullType) String() string { return nullTypeString }

// AnyType is used for return then type is any
var AnyType = new(anyType)

const anyTypeString = "any type"

type anyType struct{ nullComment }

func (t *anyType) App() IAppDef   { return nil }
func (t *anyType) IsSystem() bool { return false }
func (t *anyType) Kind() TypeKind { return TypeKind_Any }
func (t *anyType) QName() QName   { return QNameANY }
func (t *anyType) String() string { return anyTypeString }

// Is data kind allowed.
func (k TypeKind) DataKindAvailable(d DataKind) bool {
	return typeKindProps[k].fieldKinds[d]
}

// Is specified system field exists and required.
func (k TypeKind) HasSystemField(f string) (exists, required bool) {
	required, exists = typeKindProps[k].systemFields[f]
	return exists, required
}

// Is specified type kind may be used in child containers.
func (k TypeKind) ContainerKindAvailable(s TypeKind) bool {
	return typeKindProps[k].containerKinds[s]
}

func (k TypeKind) MarshalText() ([]byte, error) {
	var s string
	if k < TypeKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an TypeKind in human-readable form, without `TypeKind_` prefix,
// suitable for debugging or error messages
func (k TypeKind) TrimString() string {
	const pref = "TypeKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Type kind properties
var typeKindProps = map[TypeKind]struct {
	fieldKinds     map[DataKind]bool
	systemFields   map[string]bool
	containerKinds map[TypeKind]bool
}{
	TypeKind_null: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
	TypeKind_GDoc: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:       true,
			SystemField_QName:    true,
			SystemField_IsActive: false, // exists, but not required
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_GRecord: true,
		},
	},
	TypeKind_CDoc: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:       true,
			SystemField_QName:    true,
			SystemField_IsActive: false,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_CRecord: true,
		},
	},
	TypeKind_ODoc: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:    true,
			SystemField_QName: true,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_ODoc:    true, // #19322!: ODocs should be able to contain ODocs
			TypeKind_ORecord: true,
		},
	},
	TypeKind_WDoc: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:       true,
			SystemField_QName:    true,
			SystemField_IsActive: false,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_WRecord: true,
		},
	},
	TypeKind_GRecord: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:        true,
			SystemField_QName:     true,
			SystemField_ParentID:  true,
			SystemField_Container: true,
			SystemField_IsActive:  false,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_GRecord: true,
		},
	},
	TypeKind_CRecord: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:        true,
			SystemField_QName:     true,
			SystemField_ParentID:  true,
			SystemField_Container: true,
			SystemField_IsActive:  false,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_CRecord: true,
		},
	},
	TypeKind_ORecord: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:        true,
			SystemField_QName:     true,
			SystemField_ParentID:  true,
			SystemField_Container: true,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_ORecord: true,
		},
	},
	TypeKind_WRecord: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_ID:        true,
			SystemField_QName:     true,
			SystemField_ParentID:  true,
			SystemField_Container: true,
			SystemField_IsActive:  false,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_WRecord: true,
		},
	},
	TypeKind_ViewRecord: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
			DataKind_Record:   true,
			DataKind_Event:    true,
		},
		systemFields: map[string]bool{
			SystemField_QName: true,
		},
		containerKinds: map[TypeKind]bool{},
	},
	TypeKind_Object: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true,
			DataKind_string:   true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			SystemField_QName:     true,
			SystemField_Container: false, // exists, but required for nested (child) objects only
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_Object: true,
		},
	},
	TypeKind_Query: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
	TypeKind_Command: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
	TypeKind_Workspace: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
}
