/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SchemasCache_Add(t *testing.T) {
	require := require.New(t)

	cache := newSchemaCache()

	t.Run("panic if name is empty", func(t *testing.T) {
		require.Panics(func() {
			cache.Add(NullQName, SchemaKind_CDoc)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		require.Panics(func() {
			cache.Add(NewQName("naked", "🔫"), SchemaKind_CDoc)
		})
	})

	t.Run("if schema with name already exists", func(t *testing.T) {
		testName := NewQName("test", "test")
		cache.Add(testName, SchemaKind_CDoc)
		require.Panics(func() {
			cache.Add(testName, SchemaKind_CDoc)
		})
	})
}
