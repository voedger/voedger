/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Containers(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workapse")
	docName := appdef.NewQName("test", "doc")
	recName := appdef.NewQName("test", "rec")

	var app appdef.IAppDef

	t.Run("Should be ok to create doc with containers", func(t *testing.T) {
		adb := builder.New()
		wsb := adb.AddWorkspace(wsName)
		doc := wsb.AddWDoc(docName)
		_ = wsb.AddWRecord(recName)
		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded, "comment")

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("Should be ok to read doc with containers", func(t *testing.T) {
		doc := appdef.WDoc(app.Type, docName)
		require.NotNil(doc)

		cont := doc.Container("rec")
		require.NotNil(cont)
		require.Equal("rec", cont.Name())
		require.Equal(recName, cont.QName())
		require.EqualValues(0, cont.MinOccurs())
		require.EqualValues(appdef.Occurs_Unbounded, cont.MaxOccurs())
		require.Equal("container «rec: test.rec»", fmt.Sprint(cont))

		rec := appdef.Structure(app.Type, recName)
		require.NotNil(rec)
		require.Equal(rec, cont.Type())

		require.Equal(1, doc.ContainerCount())

		require.Equal([]appdef.IContainer{cont}, doc.Containers())
	})
}

func Test_ContainersPanics(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workapse")
	docName := appdef.NewQName("test", "doc")
	recName := appdef.NewQName("test", "rec")
	objName := appdef.NewQName("test", "obj")

	t.Run("Should be panics", func(t *testing.T) {
		adb := builder.New()
		wsb := adb.AddWorkspace(wsName)
		doc := wsb.AddCDoc(docName)
		_ = wsb.AddCRecord(recName)
		_ = wsb.AddObject(objName)

		require.Panics(func() {
			doc.AddContainer(``, recName, 0, appdef.Occurs_Unbounded)
		}, require.Is(appdef.ErrMissedError), require.Has("name"),
			"if container name missed")

		require.Panics(func() {
			doc.AddContainer(`naked 🔫`, recName, 0, appdef.Occurs_Unbounded)
		}, require.Is(appdef.ErrInvalidError), require.Has("naked 🔫"),
			"if container name invalid")

		doc.AddContainer(`rec`, recName, 0, appdef.Occurs_Unbounded)
		require.Panics(func() {
			doc.AddContainer(`rec`, recName, 0, appdef.Occurs_Unbounded)
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has("rec"),
			"if container name dupe")

		require.Panics(func() {
			doc.AddContainer(`rec1`, appdef.NullQName, 0, appdef.Occurs_Unbounded)
		}, require.Is(appdef.ErrMissedError), require.Has("rec"),
			"if container type missed")

		require.Panics(func() {
			doc.AddContainer(`rec1`, recName, 0, 0)
		}, require.Is(appdef.ErrOutOfBoundsError), require.Has("max occurs"),
			"if max occurs is zero")

		require.Panics(func() {
			doc.AddContainer(`rec1`, recName, 2, 1)
		}, require.Is(appdef.ErrOutOfBoundsError), require.Has("max occurs"),
			"if max occurs less than min occurs")

		require.Panics(func() {
			doc.AddContainer(`rec1`, objName, 0, 1)
		}, require.Is(appdef.ErrInvalidError), require.HasAll(objName, "CDoc"),
			"if container type is incompatible")

		for i := 0; i < int(appdef.MaxTypeContainerCount)-1; i++ {
			doc.AddContainer(fmt.Sprintf("rec%d", i), recName, 0, appdef.Occurs_Unbounded)
		}
		require.Panics(func() {
			doc.AddContainer("last", recName, 0, appdef.Occurs_Unbounded)
		}, require.Is(appdef.ErrTooManyError), require.Has(appdef.MaxTypeContainerCount),
			"if too many containers")
	})
}

func Test_ContainersValidate(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workapse")
	docName := appdef.NewQName("test", "doc")
	recName := appdef.NewQName("test", "rec")
	objName := appdef.NewQName("test", "obj")

	t.Run("Should be validate error", func(t *testing.T) {
		t.Run("if unknown container type", func(t *testing.T) {
			adb := builder.New()
			wsb := adb.AddWorkspace(wsName)
			doc := wsb.AddCDoc(docName)
			doc.AddContainer(`rec`, recName, 0, appdef.Occurs_Unbounded)

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.HasAll(docName, "rec", recName))
		})

		t.Run("if incompatible container type", func(t *testing.T) {
			adb := builder.New()
			wsb := adb.AddWorkspace(wsName)
			doc := wsb.AddCDoc(docName)
			doc.AddContainer(`rec`, objName, 0, appdef.Occurs_Unbounded)
			_ = wsb.AddObject(objName)

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrInvalidError), require.HasAll(docName, objName))
		})
	})
}
