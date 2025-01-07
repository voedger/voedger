/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// View is a type with key and value.
type IView interface {
	IType

	// All view fields, include key and value.
	IWithFields

	// Returns full (pk + ccols) view key
	Key() IViewKey

	// Returns view value
	Value() IViewValue
}

type IViewBuilder interface {
	ITypeBuilder

	// Returns full (pk + ccols) view key builder
	Key() IViewKeyBuilder

	// Returns view value builder
	Value() IViewValueBuilder
}

type IViewsBuilder interface {
	// Adds new types for view.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddView(QName) IViewBuilder
}

// View full (pk + cc) key.
type IViewKey interface {
	// All key fields, include partition key and clustering columns.
	//
	// Partition key fields is required, clustering columns is not.
	IWithFields

	// Returns partition key
	PartKey() IViewPartKey

	// Returns clustering columns
	ClustCols() IViewClustCols
}

type IViewKeyBuilder interface {
	// Returns partition key type builder
	PartKey() IViewPartKeyBuilder

	// Returns clustering columns type builder
	ClustCols() IViewClustColsBuilder
}

// View partition key contains fields for partitioning.
// Fields for partitioning should be selected so that the size of the partition
// does not exceed 100 MB. Perfectly if it is around 10 MB.
// The size of the basket (partition) can be evaluated by the formula:
//	(ViewValue_Size + ClustCols_Size) * KeysPerPartition.
// https://opensource.com/article/20/5/apache-cassandra#:~:text=As%20a%20rule%20of%20thumb%2C,design%20supports%20desired%20cluster%20performance
type IViewPartKey interface {
	// Partition key fields.
	IWithFields

	// Unwanted type assertion stub
	IsViewPK()
}

type IViewPartKeyBuilder interface {
	// Adds partition key field.
	//
	// # Panics:
	//	- if field already exists in clustering columns or value fields,
	//	- if not fixed size data kind.
	AddField(name FieldName, kind DataKind, constraints ...IConstraint) IViewPartKeyBuilder
	AddDataField(name FieldName, dataType QName, constraints ...IConstraint) IViewPartKeyBuilder
	AddRefField(name FieldName, ref ...QName) IViewPartKeyBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name FieldName, comment ...string) IViewPartKeyBuilder
}

// Defines fields for sorting values inside partition.
type IViewClustCols interface {
	// Clustering columns fields.
	IWithFields

	// Unwanted type assertion stub
	IsViewCC()
}

type IViewClustColsBuilder interface {
	// Adds clustering columns field.
	//
	// Only last column field can be variable length.
	//
	// # Panics:
	//	- if field already exists in view;
	//	- if already contains a variable length field.
	AddField(name FieldName, kind DataKind, constraints ...IConstraint) IViewClustColsBuilder
	AddDataField(name FieldName, dataType QName, constraints ...IConstraint) IViewClustColsBuilder
	AddRefField(name FieldName, ref ...QName) IViewClustColsBuilder

	// Sets fields comment.
	// Useful for reference or verified fields, what Add×××Field has not comments
	// argument.
	//
	// # Panics:
	//   - if field not found.
	SetFieldComment(name FieldName, comment ...string) IViewClustColsBuilder
}

// View value. Like a structure, view value has fields, but has not containers and uniques.
type IViewValue interface {
	// View value fields.
	IWithFields

	// Unwanted type assertion stub
	IsViewValue()
}

type IViewValueBuilder interface {
	IFieldsBuilder
}
