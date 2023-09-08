/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// View definition. DefKind() is DefKind_ViewRecord
//
// Ref to view.go for implementation
type IView interface {
	IDef
	IComment
	IContainers

	// Returns full (pk + ccols) view key definition
	Key() IViewKey

	// Returns view value definition
	Value() IViewValue
}

type IViewBuilder interface {
	ICommentBuilder

	// Returns full (pk + ccols) view key builder
	Key() IViewKeyBuilder

	// Returns view value builder
	Value() IViewValueBuilder
}

// View full (pk + cc) key definition. DefKind() is DefKind_ViewRecordFullKey
//
// Partition key fields is required, clustering columns is not.
//
// Ref. to view.go for implementation
type IViewKey interface {
	IDef
	IFields
	IContainers

	// Returns partition key definition
	Partition() IViewPartKey

	// Returns clustering columns definition
	ClustCols() IViewClustCols
}

// View full (pk + cc) key builder.
//
// Ref. to view.go for implementation
type IViewKeyBuilder interface {
	// Returns partition key definition
	Partition() IViewPartKeyBuilder

	// Returns clustering columns definition
	ClustCols() IViewClustColsBuilder
}

// View partition key definition. DefKind() is DefKind_ViewRecordPartitionKey
//
// Ref. to view.go for implementation
type IViewPartKey interface {
	IDef
	IFields
}

// View partition key builder.
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

// View clustering columns definition. DefKind() is DefKind_ViewRecordClusteringColumns
//
// Ref. to view.go for implementation
type IViewClustCols interface {
	IDef
	IFields
}

// View clustering columns builder.
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

// View value definition. DefKind() is DefKind_ViewRecord_Value
//
// Ref. to view.go for implementation
type IViewValue interface {
	IDef
	IFields
}

type IViewValueBuilder interface {
	IFieldsBuilder
}
