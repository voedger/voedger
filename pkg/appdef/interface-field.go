/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "regexp"

// Data kind enumeration.
//
// Ref. data-kind.go for constants and methods
type DataKind uint8

// Field Verification kind.
//
// Ref. verification-king.go for constants and methods
type VerificationKind uint8

// Definitions with fields:
//	- DefKind_GDoc and DefKind_GRecord,
//	- DefKind_CDoc and DefKind_CRecord,
//	- DefKind_ODoc and DefKind_CRecord,
//	- DefKind_WDoc and DefKind_WRecord,
//	- DefKind_Object and DefKind_Element,
//	- DefKind_ViewRecord_PartitionKey, DefKind_ViewRecord_ClusteringColumns and DefKind_ViewRecord_Value
//
// Ref. to field.go for implementation
type IFields interface {
	// Finds field by name.
	//
	// Returns nil if not found.
	Field(name string) IField

	// Returns fields count
	FieldCount() int

	// Enumerates all fields in add order.
	Fields(func(IField))

	// Finds reference field by name.
	//
	// Returns nil if field is not found, or field found but it is not a reference field
	RefField(name string) IRefField

	// Enumerates all reference fields. System field (sys.ParentID) is also enumerated
	RefFields(func(IRefField))

	// Returns reference fields count. System field (sys.ParentID) is also counted
	RefFieldCount() int

	// Enumerates all fields except system
	UserFields(func(IField))

	// Returns user fields count. System fields (sys.QName, sys.ID, …) do not count
	UserFieldCount() int
}

type IFieldsBuilder interface {
	// Adds field specified name and kind.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists,
	//   - if specified data kind is not allowed by definition kind.
	AddField(name string, kind DataKind, required bool, comment ...string) IFieldsBuilder

	// Adds reference field specified name and target refs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists.
	AddRefField(name string, required bool, ref ...QName) IFieldsBuilder

	// Adds string field specified name and restricts.
	//
	// If no restrictions specified, then field has maximum length 255.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists,
	//   - if restricts are not compatible.
	AddStringField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder

	// Adds bytes field specified name and restricts.
	//
	// If no restrictions specified, then field has maximum length 255.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if field with name is already exists,
	//   - if restricts are not compatible.
	AddBytesField(name string, required bool, restricts ...IFieldRestrict) IFieldsBuilder

	// Adds verified field specified name and kind.
	//
	// # Panics:
	//   - if field name is empty,
	//   - if field name is invalid,
	//   - if field with name is already exists,
	//   - if data kind is not allowed by definition kind,
	//   - if no verification kinds are specified
	//
	//Deprecated: use SetVerifiedField instead
	AddVerifiedField(name string, kind DataKind, required bool, vk ...VerificationKind) IFieldsBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name string, comment ...string) IFieldsBuilder

	// Sets verification kind for specified field.
	//
	// If not verification kinds are specified then it means that field is not verifiable.
	//
	// # Panics:
	//   - if field not found.
	SetFieldVerify(name string, vk ...VerificationKind) IFieldsBuilder
}

// Describe single field.
//
// Ref to field.go for constants and implementation
type IField interface {
	IComment

	// Returns field name
	Name() string

	// Returns data kind for field
	DataKind() DataKind

	// Returns is field required
	Required() bool

	// Returns is field verifiable
	Verifiable() bool

	// Returns is field verifiable by specified verification kind
	VerificationKind(VerificationKind) bool

	// Returns is field has fixed width data kind
	IsFixedWidth() bool

	// Returns is field system
	IsSys() bool
}

// Reference field. Describe field with DataKind_RecordID.
//
// Use Refs() to obtain list of target references.
//
// Ref. to fields.go for implementation
type IRefField interface {
	IField

	// Returns list of target references
	Refs() []QName
}

// String field. Describe field with DataKind_string.
//
// Use Restricts() to obtain field restricts for length, pattern.
//
// Ref. to fields.go for implementation
type IStringField interface {
	IField

	// Returns restricts
	Restricts() IStringFieldRestricts
}

// String or bytes field restricts
type IStringFieldRestricts interface {
	// Returns minimum length
	//
	// Returns 0 if not assigned
	MinLen() uint16

	// Returns maximum length
	//
	// Returns DefaultFieldMaxLength (255) if not assigned
	MaxLen() uint16

	// Returns pattern regular expression.
	//
	// Returns nil if not assigned
	Pattern() *regexp.Regexp
}

// Bytes field. Describe field with DataKind_bytes.
//
// Use Restricts() to obtain field restricts for length.
//
// Ref. to fields.go for implementation
type IBytesField interface {
	IField

	// Returns restricts
	Restricts() IBytesFieldRestricts
}

type IBytesFieldRestricts = IStringFieldRestricts

// Field restrict. Describe single restrict for field.
//
// Interface functions to obtain new restricts:
//
// # String fields:
//   - MinLen(uint16)
//   - MaxLen(uint16)
//   - Pattern(string)
//
// Ref. to fields-restrict.go
type IFieldRestrict interface{}
