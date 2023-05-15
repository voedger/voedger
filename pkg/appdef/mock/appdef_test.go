/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestBasicUsage(t *testing.T) {
	qNames := struct{ element, obj, view appdef.QName }{
		appdef.NewQName("test", "el"),
		appdef.NewQName("test", "obj"),
		appdef.NewQName("test", "view"),
	}

	el := NewDef(qNames.element, appdef.DefKind_Element,
		NewField("f1", appdef.DataKind_string, true),
	)
	obj := NewDef(qNames.obj, appdef.DefKind_Object,
		NewField("f2", appdef.DataKind_int64, true),
		NewVerifiedField("f3", appdef.DataKind_string, false, appdef.VerificationKind_Phone),
	)
	obj.AddContainer(NewContainer("c1", el.QName(), 0, appdef.Occurs_Unbounded))

	appDef := NewAppDef(
		el,
		obj,
	)

	view := NewView(qNames.view)
	view.
		AddPartField("pkFld", appdef.DataKind_int64).
		AddClustColumn("ccFld", appdef.DataKind_string).
		AddValueField("vFld1", appdef.DataKind_int64, true).
		AddValueField("vFld2", appdef.DataKind_string, false)

	appDef.AddView(view)
}

func TestInheritsMockingAppDef(t *testing.T) {
	def := Def{}
	app := AppDef{}
	app.
		On("Def", mock.AnythingOfType("appdef.QName")).Return(appdef.NullDef).
		On("DefByName", mock.AnythingOfType("appdef.QName")).Return(&def).
		On("DefCount").Return(0).
		On("Defs", mock.AnythingOfType("func(appdef.IDef)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(appdef.IDef))
			cb(&def)
		})

	require := require.New(t)

	require.Equal(appdef.NullDef, app.Def(appdef.NullQName))
	require.Equal(&def, app.DefByName(appdef.NullQName))
	require.Zero(app.DefCount())
	require.Equal(1, func() int {
		cnt := 0
		app.Defs(func(appdef.IDef) { cnt++ })
		return cnt
	}())
}
