/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
)

// Implements IViewBuilder interface
type viewBuilder struct {
	cache *appDef
	name  QName
	def,
	pkDef,
	ccDef,
	fkDef, // partition key + clustering columns
	valDef IDefBuilder
}

func newViewBuilder(cache *appDef, name QName) viewBuilder {
	view := viewBuilder{
		cache:  cache,
		name:   name,
		def:    cache.Add(name, DefKind_ViewRecord),
		pkDef:  cache.Add(ViewPartitionKeyDefName(name), DefKind_ViewRecord_PartitionKey),
		ccDef:  cache.Add(ViewClusteringColumsDefName(name), DefKind_ViewRecord_ClusteringColumns),
		fkDef:  cache.Add(ViewFullKeyColumsDefName(name), DefKind_ViewRecord_ClusteringColumns),
		valDef: cache.Add(ViewValueDefName(name), DefKind_ViewRecord_Value),
	}
	view.def.
		AddContainer(SystemContainer_ViewPartitionKey, view.pkDef.QName(), 1, 1).
		AddContainer(SystemContainer_ViewClusteringCols, view.ccDef.QName(), 1, 1).
		AddContainer(SystemContainer_ViewValue, view.valDef.QName(), 1, 1)

	return view
}

func (view *viewBuilder) AddPartField(name string, kind DataKind) ViewBuilder {
	view.pkDef.AddField(name, kind, true)
	return view
}

func (view *viewBuilder) AddClustColumn(name string, kind DataKind) ViewBuilder {
	view.ccDef.AddField(name, kind, false)
	return view
}

func (view *viewBuilder) AddValueField(name string, kind DataKind, required bool) ViewBuilder {
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

func (view *viewBuilder) FullKeyDef() IDefBuilder {
	if view.fkDef.FieldCount() != view.PartKeyDef().FieldCount()+view.ClustColsDef().FieldCount() {
		view.fkDef.clear()
		view.PartKeyDef().Fields(func(f Field) {
			view.fkDef.AddField(f.Name(), f.DataKind(), true)
		})
		view.ClustColsDef().Fields(func(f Field) {
			view.fkDef.AddField(f.Name(), f.DataKind(), false)
		})
	}
	return view.fkDef
}

func (view *viewBuilder) ValueDef() IDefBuilder {
	return view.valDef
}

func (app *appDef) prepareViewFullKeyDef(sch IDef) {
	if sch.Kind() != DefKind_ViewRecord {
		panic(fmt.Errorf("definition «%v» kind «%v» is not view: %w", sch.QName(), sch.Kind(), ErrInvalidDefKind))
	}

	contDef := func(name string, expectedKind DefKind) IDef {
		cd := sch.ContainerDef(name)
		if cd == nil {
			return nil
		}
		if cd.Kind() != expectedKind {
			return nil
		}
		return cd
	}

	pkDef := contDef(SystemContainer_ViewPartitionKey, DefKind_ViewRecord_PartitionKey)
	if pkDef == nil {
		return
	}
	ccDef := contDef(SystemContainer_ViewClusteringCols, DefKind_ViewRecord_ClusteringColumns)
	if ccDef == nil {
		return
	}

	fkName := ViewFullKeyColumsDefName(sch.QName())
	var fkDef IDefBuilder
	fkDef, ok := app.defs[fkName]

	if ok {
		if fkDef.Kind() != DefKind_ViewRecord_ClusteringColumns {
			panic(fmt.Errorf("definition «%v» has unvalid kind «%v», expected kind «%v»: %w", fkName, fkDef.Kind(), DefKind_ViewRecord_ClusteringColumns, ErrInvalidDefKind))
		}
		if fkDef.FieldCount() == pkDef.FieldCount()+ccDef.FieldCount() {
			return // already exists definition is ok
		}
		fkDef.clear()
	} else {
		fkDef = app.Add(fkName, DefKind_ViewRecord_ClusteringColumns)
	}

	// recreate full key definition fields
	pkDef.Fields(func(f Field) {
		fkDef.AddField(f.Name(), f.DataKind(), true)
	})
	ccDef.Fields(func(f Field) {
		fkDef.AddField(f.Name(), f.DataKind(), false)
	})

	app.changed()
}

// Returns partition key definition name for specified view
func ViewPartitionKeyDefName(view QName) QName {
	const suff = "_PartitionKey"
	return suffixedQName(view, suff)
}

// Returns clustering columns definition name for specified view
func ViewClusteringColumsDefName(view QName) QName {
	const suff = "_ClusteringColumns"
	return suffixedQName(view, suff)
}

// Returns full key definition name for specified view
func ViewFullKeyColumsDefName(view QName) QName {
	const suff = "_FullKey"
	return suffixedQName(view, suff)
}

// Returns value definition name for specified view
func ViewValueDefName(view QName) QName {
	const suff = "_Value"
	return suffixedQName(view, suff)
}

// Appends suffix to QName entity name and returns new QName
func suffixedQName(q QName, suff string) QName {
	return NewQName(q.Pkg(), q.Entity()+suff)
}
