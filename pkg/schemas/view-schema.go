/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

// ViewSchema service view schema struct.
//
// View consists from next schemas:
//   - view schema,
//   - partition key schema,
//   - clustering columns schema,
//   - full key schema and
//   - value schema
//
// Implements IViewBuilder interface
type ViewSchema struct {
	cache *SchemasCache
	name  QName
	viewSchema,
	partSchema,
	clustSchema,
	keySchema, // partition key + clustering columns
	valueSchema *Schema
}

func newViewSchema(cache *SchemasCache, name QName) ViewSchema {
	view := ViewSchema{
		cache:       cache,
		name:        name,
		viewSchema:  cache.Add(name, istructs.SchemaKind_ViewRecord),
		partSchema:  cache.Add(ViewPartitionKeySchemaName(name), istructs.SchemaKind_ViewRecord_PartitionKey),
		clustSchema: cache.Add(ViewClusteringColumsSchemaName(name), istructs.SchemaKind_ViewRecord_ClusteringColumns),
		keySchema:   cache.Add(ViewFullKeyColumsSchemaName(name), istructs.SchemaKind_ViewRecord_ClusteringColumns),
		valueSchema: cache.Add(ViewValueSchemaName(name), istructs.SchemaKind_ViewRecord_Value),
	}
	view.viewSchema.
		AddContainer(istructs.SystemContainer_ViewPartitionKey, view.partSchema.QName(), 1, 1).
		AddContainer(istructs.SystemContainer_ViewClusteringCols, view.clustSchema.QName(), 1, 1).
		AddContainer(istructs.SystemContainer_ViewValue, view.valueSchema.QName(), 1, 1)

	return view
}

// AddPartField adds specisified field to view partition key schema. Fields is always required
func (view *ViewSchema) AddPartField(name string, kind DataKind) *ViewSchema {
	view.partSchema.AddField(name, kind, true)
	return view
}

// AddClustColumn adds specisified field to view clustering columns schema. Fields is optional
func (view *ViewSchema) AddClustColumn(name string, kind DataKind) *ViewSchema {
	view.clustSchema.AddField(name, kind, false)
	return view
}

// AddValueField adds specisified field to view value schema
func (view *ViewSchema) AddValueField(name string, kind DataKind, required bool) *ViewSchema {
	view.ValueSchema().AddField(name, kind, required)
	return view
}

// Name returns view name
func (view *ViewSchema) Name() QName {
	return view.name
}

// Schema returns view schema
func (view *ViewSchema) Schema() *Schema {
	return view.viewSchema
}

// PartKeySchema: returns view partition key schema
func (view *ViewSchema) PartKeySchema() *Schema {
	return view.partSchema
}

// ClustColsSchema returns view clustering columns schema
func (view *ViewSchema) ClustColsSchema() *Schema {
	return view.clustSchema
}

// FullKeySchema returns view full key (partition key + clustering columns) schema
func (view *ViewSchema) FullKeySchema() *Schema {
	if view.keySchema.FieldCount() != view.PartKeySchema().FieldCount()+view.ClustColsSchema().FieldCount() {
		view.keySchema.clearFields()
		view.PartKeySchema().Fields(func(name string, kind DataKind) {
			view.keySchema.AddField(name, kind, false)
		})
		view.ClustColsSchema().Fields(func(name string, kind DataKind) {
			view.keySchema.AddField(name, kind, false)
		})
	}
	return view.keySchema
}

// ValueSchema returns view value schema
func (view *ViewSchema) ValueSchema() *Schema {
	return view.valueSchema
}

func (cache *SchemasCache) prepareViewFullKeySchema(sch *Schema) {
	if sch.Kind() != istructs.SchemaKind_ViewRecord {
		panic(fmt.Errorf("not view schema «%v» kind «%v» passed: %w", sch.QName(), sch.Kind(), ErrInvalidSchemaKind))
	}

	contSchema := func(name string, expectedKind SchemaKind) *Schema {
		contSchema := sch.ContainerSchema(name)
		if contSchema == nil {
			return nil
		}
		if contSchema.Kind() != expectedKind {
			return nil
		}
		return contSchema
	}

	pkSchema := contSchema(istructs.SystemContainer_ViewPartitionKey, istructs.SchemaKind_ViewRecord_PartitionKey)
	if pkSchema == nil {
		return
	}
	ccSchema := contSchema(istructs.SystemContainer_ViewClusteringCols, istructs.SchemaKind_ViewRecord_ClusteringColumns)
	if ccSchema == nil {
		return
	}

	fkName := ViewFullKeyColumsSchemaName(sch.QName())
	fkSchema := cache.SchemaByName(fkName)
	if fkSchema == nil {
		fkSchema = cache.Add(fkName, istructs.SchemaKind_ViewRecord_ClusteringColumns)
	} else {
		if fkSchema.Kind() != istructs.SchemaKind_ViewRecord_ClusteringColumns {
			panic(fmt.Errorf("schema «%v» has unvalid kind «%v», expected kind «%v»: %w", fkName, fkSchema.Kind(), istructs.SchemaKind_ViewRecord_ClusteringColumns, ErrInvalidSchemaKind))
		}
	}

	if fkSchema.FieldCount() == pkSchema.FieldCount()+ccSchema.FieldCount() {
		return // already updated
	}

	fkSchema.clearFields()

	pkSchema.EnumFields(func(f *Field) {
		fkSchema.AddField(f.Name(), f.DataKind(), true)
	})
	ccSchema.EnumFields(func(f *Field) {
		fkSchema.AddField(f.Name(), f.DataKind(), false)
	})
}
