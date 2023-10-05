/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// View type.
//
// Ref to view.go for implementation
type IView interface {
	IType

	// All view fields, include key and value.
	IFields

	// Returns full (pk + ccols) view key
	Key() IViewKey

	// Returns view value
	Value() IViewValue
}

type IViewBuilder interface {
	ITypeBuilder

	// Returns full (pk + ccols) view key builder
	KeyBuilder() IViewKeyBuilder

	// Returns view value builder
	ValueBuilder() IViewValueBuilder
}

// View full (pk + cc) key.
//
// Ref. to view.go for implementation
type IViewKey interface {
	// All key fields, include partition key and clustering columns.
	//
	// Partition key fields is required, clustering columns is not.
	IFields

	// Returns partition key
	PartKey() IViewPartKey

	// Returns clustering columns
	ClustCols() IViewClustCols
}

// View full (pk + cc) key builder.
//
// Ref. to view.go for implementation
type IViewKeyBuilder interface {
	// Returns partition key type builder
	PartKeyBuilder() IViewPartKeyBuilder

	// Returns clustering columns type builder
	ClustColsBuilder() IViewClustColsBuilder
}

// View partition key.
//
// Ref. to view.go for implementation
type IViewPartKey interface {
	// Partition key fields.
	IFields

	// Unwanted type assertion stub
	IsPartKey() bool
}

// View partition key type builder.
//
// Ref. to view.go for implementation
type IViewPartKeyBuilder interface {
	// Adds partition key field.
	//
	// # Panics:
	//	- if field already exists in clustering columns or value fields,
	//	- if not fixed size data kind.
	AddField(name string, kind DataKind, comment ...string) IViewPartKeyBuilder
	AddRefField(name string, ref ...QName) IViewPartKeyBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name string, comment ...string) IViewPartKeyBuilder
}

// View clustering columns.
//
// Ref. to view.go for implementation
type IViewClustCols interface {
	// Clustering columns fields.
	IFields

	// Unwanted type assertion stub
	IsClustCols() bool
}

// View clustering columns type builder.
//
// Ref. to view.go for implementation
type IViewClustColsBuilder interface {
	// Adds clustering columns field.
	//
	// Only last column field can be variable length.
	//
	// # Panics:
	//	- if field already exists in view;
	//	- if already contains a variable length field.
	AddField(name string, kind DataKind, comment ...string) IViewClustColsBuilder
	AddRefField(name string, ref ...QName) IViewClustColsBuilder
	AddStringField(name string, maxLen uint16) IViewClustColsBuilder
	AddBytesField(name string, maxLen uint16) IViewClustColsBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name string, comment ...string) IViewClustColsBuilder
}

// View value.
//
// Ref. to view.go for implementation
type IViewValue interface {
	// View value fields.
	IFields

	// Unwanted type assertion stub
	IsViewValue() bool
}

// View value builder.
//
// Ref. to view.go for implementation
type IViewValueBuilder interface {
	IFieldsBuilder
}
