/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

// TODO: this types must moved here from istructs package
type (
	QName      = istructs.QName
	SchemaKind = istructs.SchemaKindType
	DataKind   = istructs.DataKindType
	Occurs     = istructs.ContainerOccursType
)

// Application schemas cache
type SchemasCache struct {
	schemas map[QName]*Schema
}

// Schema describes the entity, such as document, record or view. Schema has fields and containers.
//
// Implements istructs.ISchema interface
type Schema struct {
	cache             *SchemasCache
	name              QName
	kind              SchemaKind
	props             SchemaKindProps
	fields            map[string]*Field
	fieldsOrdered     []string
	containers        map[string]*Container
	containersOrdered []string
	singleton         bool
}

// ViewSchema service view schema struct.
//
// View consists from next schemas:
//   - view schema,
//   - partition key schema,
//   - clustering columns schema,
//   - full key schema and
//   - value schema
type ViewSchema struct {
	cache *SchemasCache
	name  QName
	viewSchema,
	partSchema,
	clustSchema,
	keySchema, // partition key + clustering columns
	valueSchema *Schema
}

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
