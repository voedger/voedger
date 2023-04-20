/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

// Implements ISchema and ISchemaBuilder interfaces
type schemasCache struct {
	schemas map[QName]*schema
}

func newSchemaCache() *schemasCache {
	cache := schemasCache{
		schemas: make(map[QName]*schema),
	}
	return &cache
}

func (cache *schemasCache) Add(name QName, kind SchemaKind) SchemaBuilder {
	if name == istructs.NullQName {
		panic(fmt.Errorf("schema name cannot be empty: %w", ErrNameMissed))
	}
	if cache.SchemaByName(name) != nil {
		panic(fmt.Errorf("schema name «%s» already used: %w", name, ErrNameUniqueViolation))
	}
	schema := newSchema(cache, name, kind)
	cache.schemas[name] = schema
	return schema
}

func (cache *schemasCache) AddView(name QName) ViewBuilder {
	v := newViewBuilder(cache, name)
	return &v
}

func (cache *schemasCache) Build() (result SchemaCache, err error) {
	cache.prepare()

	validator := newValidator()
	cache.EnumSchemas(func(schema Schema) {
		err = errors.Join(err, validator.validate(schema))
	})
	if err != nil {
		return nil, err
	}

	return cache, nil
}

func (cache *schemasCache) SchemaByName(name QName) Schema {
	if schema, ok := cache.schemas[name]; ok {
		return schema
	}
	return nil
}

func (cache *schemasCache) SchemaCount() int {
	return len(cache.schemas)
}

func (cache *schemasCache) EnumSchemas(enum func(Schema)) {
	for _, schema := range cache.schemas {
		enum(schema)
	}
}

func (cache *schemasCache) prepare() {
	cache.EnumSchemas(func(s Schema) {
		if s.Kind() == istructs.SchemaKind_ViewRecord {
			cache.prepareViewFullKeySchema(s)
		}
	})
}

//————— istructs.ISchemas —————

func (cache *schemasCache) Schema(name istructs.QName) istructs.ISchema {
	if schema, ok := cache.schemas[name]; ok {
		return schema
	}
	return NullSchema
}

func (cache *schemasCache) Schemas(cb func(schemaName istructs.QName)) {
	cache.EnumSchemas(func(s Schema) { cb(s.QName()) })
}
