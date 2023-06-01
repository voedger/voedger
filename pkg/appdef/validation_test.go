/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateContainer(t *testing.T) {
	require := require.New(t)

	app := New()
	doc := app.AddCDoc(NewQName("test", "doc"))
	doc.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)

	t.Run("must be error if container def not found", func(t *testing.T) {
		_, err := app.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, "unknown definition «test.rec»")
	})

	rec := app.AddCRecord(NewQName("test", "rec"))
	_, err := app.Build()
	require.NoError(err)

	t.Run("must be ok container recurse", func(t *testing.T) {
		rec.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be ok container sub recurse", func(t *testing.T) {
		rec.AddContainer("rec1", NewQName("test", "rec1"), 0, Occurs_Unbounded)
		rec1 := app.AddCRecord(NewQName("test", "rec1"))
		rec1.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})
}

func TestValidateRefFields(t *testing.T) {
	require := require.New(t)

	app := New()
	doc := app.AddCDoc(NewQName("test", "doc"))
	doc.AddRefField("f1", true, NewQName("test", "rec"))

	rec := app.AddCRecord(NewQName("test", "rec"))
	rec.AddRefField("f1", true, NewQName("test", "rec"))

	t.Run("must be ok if all reference field is valid", func(t *testing.T) {
		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be error if reference field ref is not found", func(t *testing.T) {
		rec.AddRefField("f2", true, NewQName("test", "obj"))
		_, err := app.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, "unknown definition «test.obj»")
	})

	t.Run("must be error if reference field refs to non referable definition", func(t *testing.T) {
		app.AddObject(NewQName("test", "obj"))
		_, err := app.Build()
		require.ErrorIs(err, ErrInvalidDefKind)
		require.ErrorContains(err, "non referable definition «test.obj»")
	})
}
