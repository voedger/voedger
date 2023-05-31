/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	require := require.New(t)

	app := New()
	d := app.AddCDoc(NewQName("test", "cDoc"))
	d.AddContainer("rec", NewQName("test", "cRec"), 0, Occurs_Unbounded)

	t.Run("must be error if container def not found", func(t *testing.T) {
		_, err := app.Build()
		require.ErrorIs(err, ErrNameNotFound)
	})

	c := app.AddCRecord(NewQName("test", "cRec"))
	_, err := app.Build()
	require.NoError(err)

	t.Run("must be ok container recurse", func(t *testing.T) {
		c.AddContainer("rec", NewQName("test", "cRec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be ok container sub recurse", func(t *testing.T) {
		c.AddContainer("rec1", NewQName("test", "cRec1"), 0, Occurs_Unbounded)
		c1 := app.AddCRecord(NewQName("test", "cRec1"))
		c1.AddContainer("rec", NewQName("test", "cRec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})
}
