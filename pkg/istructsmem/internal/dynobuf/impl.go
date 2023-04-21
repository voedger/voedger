/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/schemas"
)

func newSchemasCache() DynoBufSchemasCache {
	cache := DynoBufSchemasCache{}
	return cache
}

// Prepares schemas
func (cache DynoBufSchemasCache) Prepare(sch schemas.SchemaCache) {
	sch.EnumSchemas(
		func(schema schemas.Schema) {
			cache.add(schema)
		})
}

// Adds schema
func (cache DynoBufSchemasCache) add(schema schemas.Schema) {
	db := dynobuffers.NewScheme()

	db.Name = schema.QName().String()
	schema.EnumFields(
		func(f schemas.Field) {
			if !f.IsSys() { // #18142: extract system fields from dynobuffer
				fieldType := DataKindToFieldType(f.DataKind())
				if fieldType == dynobuffers.FieldTypeByte {
					db.AddArray(f.Name(), fieldType, false)
				} else {
					db.AddField(f.Name(), fieldType, false)
				}
			}
		})

	cache[schema.QName()] = db
}
