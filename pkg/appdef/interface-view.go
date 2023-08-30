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
	IView
	ICommentBuilder

	// AddPartField adds specified field to view partition key definition. Fields is always required
	//
	// # Panics:
	//	- if field already exists in clustering columns or value fields,
	//	- if not fixed size data kind.
	AddPartField(name string, kind DataKind, comment ...string) IViewBuilder

	// AddClustColumn adds specified field to view clustering columns definition. Fields is optional
	//
	// # Panics:
	//	- if field already exists in partition key or value fields.
	AddClustColumn(name string, kind DataKind, comment ...string) IViewBuilder

	// AddValueField adds specified field to view value definition
	//
	// # Panics:
	//	- if field already exists in partition key or clustering columns fields.
	AddValueField(name string, kind DataKind, required bool, comment ...string) IViewBuilder
}

// View partition key definition. DefKind() is DefKind_ViewRecordPartitionKey
//
// Ref. to view.go for implementation
type IPartKey interface {
	IDef
	IComment
	IFields
}

// View clustering columns definition. DefKind() is DefKind_ViewRecordClusteringColumns
//
// Ref. to view.go for implementation
type IClustCols interface {
	IDef
	IComment
	IFields
}

// View full (pk + cc) key definition. DefKind() is DefKind_ViewRecordFullKey
//
// Partition key fields is required, clustering columns is not.
//
// Ref. to view.go for implementation
type IViewKey interface {
	IDef
	IComment
	IFields
	IContainers

	// Returns partition key definition
	Partition() IPartKey

	// Returns clustering columns definition
	ClustCols() IClustCols
}

// View value definition. DefKind() is DefKind_ViewRecord_Value
//
// Ref. to view.go for implementation
type IViewValue interface {
	IDef
	IComment
	IFields
}
