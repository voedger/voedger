/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"fmt"

	"github.com/untillpro/voedger/pkg/istructs"
)

func newSchemaCache() *SchemasCache {
	cache := SchemasCache{
		schemas: make(map[QName]*Schema),
	}
	return &cache
}

// Adds new schema specified name and kind.
//
// # Panics:
//   - if name is empty (istructs.NullQName),
//   - if schema with name already exists.
func (cache *SchemasCache) Add(name QName, kind SchemaKind) (schema *Schema) {
	if name == istructs.NullQName {
		panic(fmt.Errorf("schema name cannot be empty: %w", ErrNameMissed))
	}
	if cache.SchemaByName(name) != nil {
		panic(fmt.Errorf("schema name «%s» already used: %w", name, ErrNameUniqueViolation))
	}
	schema = newSchema(cache, name, kind)
	cache.schemas[name] = schema
	return schema
}

// Adds new schemas for view.
func (cache *SchemasCache) AddView(name QName) *ViewSchema {
	v := newViewSchema(cache, name)
	return &v
}

// Enumerates all schemas from cache.
func (cache *SchemasCache) EnumSchemas(enum func(*Schema)) {
	for _, schema := range cache.schemas {
		enum(schema)
	}
}

// Returns schema by name.
//
// Returns nil if not found.
func (cache *SchemasCache) SchemaByName(name QName) *Schema {
	if schema, ok := cache.schemas[name]; ok {
		return schema
	}
	return nil
}

// Prepares cache for use. Automaticaly called from ValidateSchemas method.
func (cache *SchemasCache) Prepare() {
	cache.EnumSchemas(func(s *Schema) {
		if s.Kind() == istructs.SchemaKind_ViewRecord {
			cache.prepareViewFullKeySchema(s)
		}
	})
}

// —————————— istructs.ISchemas ——————————

func (cache *SchemasCache) Schema(schema QName) istructs.ISchema {
	s := cache.SchemaByName(schema)
	if s == nil {
		return NullSchema
	}
	return s
}

func (cache *SchemasCache) Schemas(enum func(QName)) {
	cache.EnumSchemas(func(schema *Schema) { enum(schema.QName()) })
}
