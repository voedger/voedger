/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package vers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

func Test_BasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(mem.Provide(testingu.MockTime))
	storage, err := sp.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(t, err)

	versions := New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	t.Run("basic Versions methods", func(t *testing.T) {
		require := require.New(t)

		const (
			key VersionKey   = 55
			ver VersionValue = 77
		)

		require.Equal(UnknownVersion, versions.Get(key))

		versions.Put(key, ver)
		require.Equal(ver, versions.Get(key))

		t.Run("must be able to load early stored versions", func(t *testing.T) {
			otherVersions := New()
			if err := otherVersions.Prepare(storage); err != nil {
				panic(err)
			}
			require.Equal(ver, otherVersions.Get(key))
		})
	})

}
