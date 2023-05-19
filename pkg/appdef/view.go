/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// Implements IViewBuilder interface
type viewBuilder struct {
	cache *appDef
	name  QName
	def,
	pkDef,
	ccDef,
	valDef IDefBuilder
}

func newViewBuilder(cache *appDef, name QName) viewBuilder {
	view := viewBuilder{
		cache:  cache,
		name:   name,
		def:    cache.addDef(name, DefKind_ViewRecord),
		pkDef:  cache.addDef(ViewPartitionKeyDefName(name), DefKind_ViewRecord_PartitionKey),
		ccDef:  cache.addDef(ViewClusteringColumnsDefName(name), DefKind_ViewRecord_ClusteringColumns),
		valDef: cache.addDef(ViewValueDefName(name), DefKind_ViewRecord_Value),
	}
	view.def.
		AddContainer(SystemContainer_ViewPartitionKey, view.pkDef.QName(), 1, 1).
		AddContainer(SystemContainer_ViewClusteringCols, view.ccDef.QName(), 1, 1).
		AddContainer(SystemContainer_ViewValue, view.valDef.QName(), 1, 1)

	return view
}

func (view *viewBuilder) AddPartField(name string, kind DataKind) IViewBuilder {
	view.panicIfFieldDuplication(name)
	view.pkDef.AddField(name, kind, true)
	return view
}

func (view *viewBuilder) AddClustColumn(name string, kind DataKind) IViewBuilder {
	view.panicIfFieldDuplication(name)
	view.ccDef.AddField(name, kind, false)
	return view
}

func (view *viewBuilder) AddValueField(name string, kind DataKind, required bool) IViewBuilder {
	view.panicIfFieldDuplication(name)
	view.ValueDef().AddField(name, kind, required)
	return view
}

func (view *viewBuilder) Def() IDefBuilder {
	return view.def
}

func (view *viewBuilder) Name() QName {
	return view.name
}

func (view *viewBuilder) PartKeyDef() IDefBuilder {
	return view.pkDef
}

func (view *viewBuilder) ClustColsDef() IDefBuilder {
	return view.ccDef
}

func (view *viewBuilder) ValueDef() IDefBuilder {
	return view.valDef
}

func (view *viewBuilder) panicIfFieldDuplication(name string) {
	check := func(def IDef) {
		if def.Field(name) != nil {
			panic(fmt.Errorf("field «%s» already exists in view «%v» %v: %w", name, view.Name(), def.Kind(), ErrNameUniqueViolation))
		}
	}

	check(view.PartKeyDef())
	check(view.ClustColsDef())
	check(view.ValueDef())
}

func (app *appDef) prepareViewFullKeyDef(def IDef) {
	pkDef := def.ContainerDef(SystemContainer_ViewPartitionKey)
	ccDef := def.ContainerDef(SystemContainer_ViewClusteringCols)

	fkName := ViewFullKeyColumnsDefName(def.QName())
	var fkDef IDefBuilder
	fkDef, ok := app.defs[fkName]

	if ok {
		if (fkDef.Kind() == DefKind_ViewRecord_ClusteringColumns) &&
			(fkDef.FieldCount() == pkDef.FieldCount()+ccDef.FieldCount()) {
			return // already exists definition is ok
		}
		app.remove(fkName)
	}

	// recreate full key definition fields
	fkDef = app.addDef(fkName, DefKind_ViewRecord_ClusteringColumns)

	pkDef.Fields(func(f IField) {
		fkDef.AddField(f.Name(), f.DataKind(), true)
	})
	ccDef.Fields(func(f IField) {
		fkDef.AddField(f.Name(), f.DataKind(), false)
	})

	app.changed()
}

// Returns partition key definition name for specified view
func ViewPartitionKeyDefName(view QName) QName {
	const suffix = "_PartitionKey"
	return suffixedQName(view, suffix)
}

// Returns clustering columns definition name for specified view
func ViewClusteringColumnsDefName(view QName) QName {
	const suffix = "_ClusteringColumns"
	return suffixedQName(view, suffix)
}

// Returns full key definition name for specified view
func ViewFullKeyColumnsDefName(view QName) QName {
	const suffix = "_FullKey"
	return suffixedQName(view, suffix)
}

// Returns value definition name for specified view
func ViewValueDefName(view QName) QName {
	const suffix = "_Value"
	return suffixedQName(view, suffix)
}

// Appends suffix to QName entity name and returns new QName
func suffixedQName(q QName, suffix string) QName {
	return NewQName(q.Pkg(), q.Entity()+suffix)
}
