/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/objcache"
	"github.com/voedger/voedger/pkg/objcache/internal/test"
)

func Test_BasicUsage(t *testing.T) {
	test.TechnologyCompatibilityKit(t, objcache.New[test.IOffset, test.IEvent])
}

func Test_Providers(t *testing.T) {
	for p := objcache.Hashicorp; p < objcache.CacheProvider_Count; p++ {
		new := func(size int, onEvicted func(test.IOffset, test.IEvent)) test.TckCache {
			return objcache.NewProvider[test.IOffset, test.IEvent](p, size, onEvicted)
		}
		test.TechnologyCompatibilityKit(t, new)
	}

	t.Run("must be panic if unknown cache provider", func(t *testing.T) {
		require.New(t).Panics(func() { objcache.NewProvider[int, string](objcache.CacheProvider_Count, 1, nil) })
	})
}
