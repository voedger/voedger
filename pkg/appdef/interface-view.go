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
	IComment
	IContainers

	// Returns full (pk + ccols) view key type
	Key() IViewKey

	// Returns view value type
	Value() IViewValue
}

type IViewBuilder interface {
	ICommentBuilder

	// Returns full (pk + ccols) view key builder
	Key() IViewKeyBuilder

	// Returns view value builder
	Value() IViewValueBuilder
}

// View full (pk + cc) key type.
//
// Partition key fields is required, clustering columns is not.
//
// Ref. to view.go for implementation
type IViewKey interface {
	IType
	IFields
	IContainers

	// Returns partition key type
	Partition() IViewPartKey

	// Returns clustering columns type
	ClustCols() IViewClustCols
}

// View full (pk + cc) key builder.
//
// Ref. to view.go for implementation
type IViewKeyBuilder interface {
	// Returns partition key type builder
	Partition() IViewPartKeyBuilder

	// Returns clustering columns type builder
	ClustCols() IViewClustColsBuilder
}

// View partition key type.
//
// Ref. to view.go for implementation
type IViewPartKey interface {
	IType
	IFields
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

// View clustering columns type.
//
// Ref. to view.go for implementation
type IViewClustCols interface {
	IType
	IFields
}

// View clustering columns type builder.
//
// Ref. to view.go for implementation
type IViewClustColsBuilder interface {
	// Adds clustering columns field.
	//
	// # Panics:
	//	- if field already exists in partition key or value fields.
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

// View value type.
//
// Ref. to view.go for implementation
type IViewValue interface {
	IType
	IFields
}

type IViewValueBuilder interface {
	IFieldsBuilder
}
