/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type View struct {
	mock.Mock
	app *AppDef
	view,
	pk,
	cc,
	val *Def
}

func NewView(name appdef.QName) *View {
	v := View{
		view: NewDef(name, appdef.DefKind_ViewRecord),
		pk:   NewDef(appdef.ViewPartitionKeyDefName(name), appdef.DefKind_ViewRecord_PartitionKey),
		cc:   NewDef(appdef.ViewClusteringColumsDefName(name), appdef.DefKind_ViewRecord_ClusteringColumns),
		val:  NewDef(appdef.ViewValueDefName(name), appdef.DefKind_ViewRecord_Value),
	}

	v.view.AddContainer(NewContainer(appdef.SystemContainer_ViewPartitionKey, v.pk.QName(), 1, 1))
	v.view.AddContainer(NewContainer(appdef.SystemContainer_ViewClusteringCols, v.cc.QName(), 1, 1))
	v.view.AddContainer(NewContainer(appdef.SystemContainer_ViewValue, v.val.QName(), 1, 1))

	return &v
}

func (v *View) AddPartField(name string, kind appdef.DataKind) *View {
	v.pk.AddField(NewField(name, kind, true))
	return v
}

func (v *View) AddClustColumn(name string, kind appdef.DataKind) *View {
	v.cc.AddField(NewField(name, kind, false))
	return v
}

func (v *View) AddValueField(name string, kind appdef.DataKind, required bool) *View {
	v.val.AddField(NewField(name, kind, required))
	return v
}
