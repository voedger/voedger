/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package types_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Tags(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	tagNames := []appdef.QName{appdef.NewQName("test", "tag1"), appdef.NewQName("test", "tag2")}
	docName := appdef.NewQName("test", "doc")

	var app appdef.IAppDef

	t.Run("should be ok to add doc with tag", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagNames[0], "first feature", "first tag comment")
		wsb.AddTag(tagNames[1], "second feature", "second tag comment")

		doc := wsb.AddODoc(docName)
		doc.AddField("f1", appdef.DataKind_int64, true)
		doc.SetTag(tagNames...)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find tag by name", func(t *testing.T) {
			tag := appdef.Tag(tested.Type, tagNames[0])
			require.NotNil(tag)
			require.Equal(tagNames[0], tag.QName())
			require.Equal(appdef.TypeKind_Tag, tag.Kind())
			require.Equal("first feature", tag.Feature())
			require.Equal("first tag comment", tag.Comment())

			require.Nil(appdef.Tag(tested.Type, appdef.NewQName("test", "unknown")), "should nil if unknown tag")
		})

		t.Run("should be ok to enumerate tags", func(t *testing.T) {
			got := appdef.QNames{}
			for tag := range appdef.Tags(tested.Types()) {
				got.Add(tag.QName())
			}
			require.Equal(appdef.QNamesFrom(tagNames...), got)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	doc := appdef.ODoc(app.Type, docName)

	t.Run("should be ok to inspect doc tags", func(t *testing.T) {
		for _, tag := range tagNames {
			require.True(doc.HasTag(tag), "should have tag")
		}
		require.False(doc.HasTag(appdef.NewQName("test", "unknown")), "should not if unknown tag")
		require.False(doc.HasTag(docName), "should not if not a tag")
	})

	t.Run("should be ok to enumerate tags", func(t *testing.T) {
		got := appdef.QNames{}
		for _, t := range doc.Tags() {
			got.Add(t.QName())
		}
		require.Equal(appdef.QNamesFrom(tagNames...), got)
	})
}

func Test_TagsPanics(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	tagNames := []appdef.QName{appdef.NewQName("test", "tag1"), appdef.NewQName("test", "tag2")}
	docName := appdef.NewQName("test", "doc")

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if set unknown tag", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			doc := wsb.AddODoc(docName)
			require.Panics(func() { doc.SetTag(tagNames[0]) })
		})

		// #2889 $VSQL_TagNonExp: only local tags can be used
		t.Run("if set tag from ancestor ws", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			ancName := appdef.NewQName("test", "wsAncestor")
			tagName := appdef.NewQName("test", "tagAncestor")

			ancWS := adb.AddWorkspace(ancName)
			ancWS.AddTag(tagName)

			wsb := adb.AddWorkspace(wsName)
			wsb.SetAncestors(ancName)

			doc := wsb.AddODoc(docName)
			require.Panics(func() { doc.SetTag(tagName) })
		})
	})
}
