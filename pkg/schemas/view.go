/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"fmt"
)

// Implements IViewBuilder interface
type viewBuilder struct {
	cache *schemasCache
	name  QName
	viewSchema,
	partSchema,
	clustSchema,
	keySchema, // partition key + clustering columns
	valueSchema SchemaBuilder
}

func newViewBuilder(cache *schemasCache, name QName) viewBuilder {
	view := viewBuilder{
		cache:       cache,
		name:        name,
		viewSchema:  cache.Add(name, SchemaKind_ViewRecord),
		partSchema:  cache.Add(ViewPartitionKeySchemaName(name), SchemaKind_ViewRecord_PartitionKey),
		clustSchema: cache.Add(ViewClusteringColumsSchemaName(name), SchemaKind_ViewRecord_ClusteringColumns),
		keySchema:   cache.Add(ViewFullKeyColumsSchemaName(name), SchemaKind_ViewRecord_ClusteringColumns),
		valueSchema: cache.Add(ViewValueSchemaName(name), SchemaKind_ViewRecord_Value),
	}
	view.viewSchema.
		AddContainer(SystemContainer_ViewPartitionKey, view.partSchema.QName(), 1, 1).
		AddContainer(SystemContainer_ViewClusteringCols, view.clustSchema.QName(), 1, 1).
		AddContainer(SystemContainer_ViewValue, view.valueSchema.QName(), 1, 1)

	return view
}

func (view *viewBuilder) AddPartField(name string, kind DataKind) ViewBuilder {
	view.partSchema.AddField(name, kind, true)
	return view
}

func (view *viewBuilder) AddClustColumn(name string, kind DataKind) ViewBuilder {
	view.clustSchema.AddField(name, kind, false)
	return view
}

func (view *viewBuilder) AddValueField(name string, kind DataKind, required bool) ViewBuilder {
	view.ValueSchema().AddField(name, kind, required)
	return view
}

func (view *viewBuilder) Name() QName {
	return view.name
}

func (view *viewBuilder) Schema() SchemaBuilder {
	return view.viewSchema
}

func (view *viewBuilder) PartKeySchema() SchemaBuilder {
	return view.partSchema
}

func (view *viewBuilder) ClustColsSchema() SchemaBuilder {
	return view.clustSchema
}

// FullKeySchema returns view full key (partition key + clustering columns) schema
func (view *viewBuilder) FullKeySchema() SchemaBuilder {
	if view.keySchema.FieldCount() != view.PartKeySchema().FieldCount()+view.ClustColsSchema().FieldCount() {
		view.keySchema.clear()
		view.PartKeySchema().EnumFields(func(fld Field) {
			view.keySchema.AddField(fld.Name(), fld.DataKind(), true)
		})
		view.ClustColsSchema().EnumFields(func(fld Field) {
			view.keySchema.AddField(fld.Name(), fld.DataKind(), false)
		})
	}
	return view.keySchema
}

func (view *viewBuilder) ValueSchema() SchemaBuilder {
	return view.valueSchema
}

func (cache *schemasCache) prepareViewFullKeySchema(sch Schema) {
	if sch.Kind() != SchemaKind_ViewRecord {
		panic(fmt.Errorf("not view schema «%v» kind «%v» passed: %w", sch.QName(), sch.Kind(), ErrInvalidSchemaKind))
	}

	contSchema := func(name string, expectedKind SchemaKind) Schema {
		contSchema := sch.ContainerSchema(name)
		if contSchema == nil {
			return nil
		}
		if contSchema.Kind() != expectedKind {
			return nil
		}
		return contSchema
	}

	pkSchema := contSchema(SystemContainer_ViewPartitionKey, SchemaKind_ViewRecord_PartitionKey)
	if pkSchema == nil {
		return
	}
	ccSchema := contSchema(SystemContainer_ViewClusteringCols, SchemaKind_ViewRecord_ClusteringColumns)
	if ccSchema == nil {
		return
	}

	fkName := ViewFullKeyColumsSchemaName(sch.QName())
	var fkSchema SchemaBuilder
	fkSchema, ok := cache.schemas[fkName]

	if ok {
		if fkSchema.Kind() != SchemaKind_ViewRecord_ClusteringColumns {
			panic(fmt.Errorf("schema «%v» has unvalid kind «%v», expected kind «%v»: %w", fkName, fkSchema.Kind(), SchemaKind_ViewRecord_ClusteringColumns, ErrInvalidSchemaKind))
		}
		if fkSchema.FieldCount() == pkSchema.FieldCount()+ccSchema.FieldCount() {
			return // already exists schema is ok
		}
		fkSchema.clear()
	} else {
		fkSchema = cache.Add(fkName, SchemaKind_ViewRecord_ClusteringColumns)
	}

	// recreate full key schema fields
	pkSchema.EnumFields(func(f Field) {
		fkSchema.AddField(f.Name(), f.DataKind(), true)
	})
	ccSchema.EnumFields(func(f Field) {
		fkSchema.AddField(f.Name(), f.DataKind(), false)
	})
}

// Returns partition key schema name for specified view
func ViewPartitionKeySchemaName(view QName) QName {
	const suff = "_PartitionKey"
	return suffixedQName(view, suff)
}

// Returns clustering columns schema name for specified view
func ViewClusteringColumsSchemaName(view QName) QName {
	const suff = "_ClusteringColumns"
	return suffixedQName(view, suff)
}

// Returns full key schema name for specified view
func ViewFullKeyColumsSchemaName(view QName) QName {
	const suff = "_FullKey"
	return suffixedQName(view, suff)
}

// Returns value schema name for specified view
func ViewValueSchemaName(view QName) QName {
	const suff = "_Value"
	return suffixedQName(view, suff)
}

// Appends suffix to QName entity name and returns new QName
func suffixedQName(q QName, suff string) QName {
	return NewQName(q.Pkg(), q.Entity()+suff)
}
