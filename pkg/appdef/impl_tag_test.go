/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestTags(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "workspace")
	tagNames := []QName{NewQName("test", "tag1"), NewQName("test", "tag2")}
	docName := NewQName("test", "doc")

	var app IAppDef

	t.Run("should be ok to add doc with tag", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagNames[0], "first tag comment")
		wsb.AddTag(tagNames[1])
		wsb.SetTypeComment(tagNames[1], "second tag comment")

		doc := wsb.AddODoc(docName)
		doc.AddField("f1", DataKind_int64, true)
		doc.SetTag(tagNames[0], tagNames[1:]...)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find tag by name", func(t *testing.T) {
			tag := Tag(tested.Type, tagNames[0])
			require.NotNil(tag)
			require.Equal(tagNames[0], tag.QName())
			require.Equal(TypeKind_Tag, tag.Kind())
			require.Equal("first tag comment", tag.Comment())
			require.Equal(wsName, tag.Workspace().QName())

			require.Nil(Tag(tested.Type, NewQName("test", "unknown")), "should nil if unknown tag")
		})

		t.Run("should be ok to enumerate tags", func(t *testing.T) {
			got := QNames{}
			for tag := range Tags(tested.Types) {
				got.Add(tag.QName())
			}
			require.Equal(QNamesFrom(tagNames...), got)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	doc := ODoc(app.Type, docName)

	t.Run("should be ok to inspect doc tags", func(t *testing.T) {
		for _, tag := range tagNames {
			require.True(doc.HasTag(tag), "should have tag")
		}
		require.False(doc.HasTag(NewQName("test", "unknown")), "should not if unknown tag")
		require.False(doc.HasTag(docName), "should not if not a tag")
	})

	t.Run("should be ok to enumerate tags", func(t *testing.T) {
		got := QNames{}
		for t := range doc.Tags {
			got.Add(t.QName())
		}
		require.Equal(QNamesFrom(tagNames...), got)
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if set unknown tag", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			doc := wsb.AddODoc(docName)
			require.Panics(func() { doc.SetTag(tagNames[0]) })
		})
	})
}
