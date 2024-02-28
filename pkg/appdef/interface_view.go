/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// View is a type with key and value.
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
	IView
	ITypeBuilder

	// Returns full (pk + ccols) view key builder
	KeyBuilder() IViewKeyBuilder

	// Returns view value builder
	ValueBuilder() IViewValueBuilder
}

// View full (pk + cc) key.
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

type IViewKeyBuilder interface {
	IViewKey

	// Returns partition key type builder
	PartKeyBuilder() IViewPartKeyBuilder

	// Returns clustering columns type builder
	ClustColsBuilder() IViewClustColsBuilder
}

// View partition key contains fields for partitioning.
// Fields for partitioning should be selected so that the size of the partition
// does not exceed 100 MB. Perfectly if it is around 10 MB.
// The size of the basket (partition) can be evaluated by the formula:
//	(ViewValue_Size + ClustCols_Size) * KeysPerPartition.
// https://opensource.com/article/20/5/apache-cassandra#:~:text=As%20a%20rule%20of%20thumb%2C,design%20supports%20desired%20cluster%20performance
type IViewPartKey interface {
	// Partition key fields.
	IFields

	// Unwanted type assertion stub
	isPartKey()
}

type IViewPartKeyBuilder interface {
	IViewPartKey

	// Adds partition key field.
	//
	// # Panics:
	//	- if field already exists in clustering columns or value fields,
	//	- if not fixed size data kind.
	AddField(name string, kind DataKind, constraints ...IConstraint) IViewPartKeyBuilder
	AddDataField(name string, dataType QName, constraints ...IConstraint) IViewPartKeyBuilder
	AddRefField(name string, ref ...QName) IViewPartKeyBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name string, comment ...string) IViewPartKeyBuilder
}

// Defines fields for sorting values inside partition.
type IViewClustCols interface {
	// Clustering columns fields.
	IFields

	// Unwanted type assertion stub
	isClustCols()
}

type IViewClustColsBuilder interface {
	IViewClustCols

	// Adds clustering columns field.
	//
	// Only last column field can be variable length.
	//
	// # Panics:
	//	- if field already exists in view;
	//	- if already contains a variable length field.
	AddField(name string, kind DataKind, constraints ...IConstraint) IViewClustColsBuilder
	AddDataField(name string, dataType QName, constraints ...IConstraint) IViewClustColsBuilder
	AddRefField(name string, ref ...QName) IViewClustColsBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name string, comment ...string) IViewClustColsBuilder
}

// View value. Like a structure, view value has fields, but has not containers and uniques.
type IViewValue interface {
	// View value fields.
	IFields

	// Unwanted type assertion stub
	isViewValue()
}

type IViewValueBuilder interface {
	IViewValue

	IFieldsBuilder
}
