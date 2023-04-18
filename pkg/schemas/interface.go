/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// TODO: this types must moved here from istructs package
type (
	QName      = istructs.QName
	SchemaKind = istructs.SchemaKindType
	DataKind   = istructs.DataKindType
	Occurs     = istructs.ContainerOccursType
)

// // Application schema cache
// type ISchemasCache interface {
// 	Schemas(func(ISchema))
// 	SchemaCount() uint
// 	Schema(name QName) ISchema
// }

// // Application schema cache builder
// type ISchemasCacheBuilder interface {
// 	ISchemasCache
// 	Add(name QName, kind SchemaKind) ISchemaBuilder
// 	AddView(QName) IViewBuilder
// }

// // Schema describes the entity, such as document, record or view. Schema has fields and containers.
// type ISchema interface {
// 	Name() QName
// 	Kind() SchemaKind

// 	Fields(func(IField))
// 	FieldCount() uint
// 	Field(name string) IField

// 	Containers(func(IContainer))
// 	ContainerCount() uint
// 	Container(name string) IContainer
// }

// // Schema builder
// type ISchemaBuilder interface {
// 	ISchema
// 	AddField(name string, kind DataKind, required bool) IField
// 	AddContainer(name string, schema QName, min, max Occurs) IContainer
// }

// // View builder
// type IViewBuilder interface {
// 	AddPartField(name string, kind DataKind) IViewBuilder
// 	AddClustColumn(name string, kind DataKind) IViewBuilder
// 	AddValueField(name string, kind DataKind, required bool) IViewBuilder
// }

// Describe single field.
//
// Implements istructs.IFieldDescr interface
type Field struct {
	name       string
	kind       DataKind
	required   bool
	verifiable bool
}

// Describes single inclusion of child schema in parent schema.
//
// Implements istructs.IContainerDesc interface
type Container struct {
	name      string
	schema    QName
	minOccurs Occurs
	maxOccurs Occurs
}
