/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package iterate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	ITested1 interface {
		Fields(enum func(name string))
	}
	tested1 struct {
		fields []string
	}
)

func (s *tested1) Fields(enum func(name string)) {
	for _, name := range s.fields {
		enum(name)
	}
}

type (
	ITested2 interface {
		Fields(enum func(name string, data int))
	}
	tested2 struct {
		fields map[string]int
	}
)

func (s *tested2) Fields(enum func(name string, data int)) {
	for name, data := range s.fields {
		enum(name, data)
	}
}

func Test_FindFirstMap(t *testing.T) {
	require := require.New(t)

	t.Run("test FindFirstMap with interface method", func(t *testing.T) {
		var tested ITested2 = &tested2{fields: map[string]int{"a": 1, "b": 2, "c": 3}}

		ok, key, value := FindFirstMap(tested.Fields, func(k string, _ int) bool { return k == "b" })
		require.True(ok)
		require.Equal("b", key)
		require.Equal(2, value)

		ok, key, value = FindFirstMap(tested.Fields, func(k string, _ int) bool { return k == "impossible" })
		require.False(ok)
		require.Empty(key)
		require.Empty(value)
	})

	t.Run("test FindFirstMap with structure method", func(t *testing.T) {
		tested := tested2{fields: map[string]int{"a": 1, "b": 2, "c": 3}}

		ok, key, value := FindFirstMap(tested.Fields, func(k string, _ int) bool { return k == "b" })
		require.True(ok)
		require.Equal("b", key)
		require.Equal(2, value)

		ok, key, value = FindFirstMap(tested.Fields, func(k string, _ int) bool { return k == "impossible" })
		require.False(ok)
		require.Empty(key)
		require.Empty(value)
	})

	t.Run("test FindFirstMap with naked map", func(t *testing.T) {
		fields := map[string]int{"a": 1, "b": 2, "c": 3}

		ok, key, value := FindFirstMap(Map(fields), func(k string, _ int) bool { return k == "b" })
		require.True(ok)
		require.Equal("b", key)
		require.Equal(2, value)

		ok, key, value = FindFirstMap(Map(fields), func(k string, _ int) bool { return k == "impossible" })
		require.False(ok)
		require.Empty(key)
		require.Empty(value)
	})
}
