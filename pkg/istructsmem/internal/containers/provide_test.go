/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func Test_BasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(mem.Provide(coreutils.MockTime))
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	testName := "test"
	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	wsb.AddObject(appdef.NewQName("test", "obj")).
		AddContainer(testName, appdef.NewQName("test", "obj"), 0, appdef.Occurs_Unbounded)
	appDef, err := adb.Build()
	if err != nil {
		panic(err)
	}

	containers := New()
	if err := containers.Prepare(storage, versions, appDef); err != nil {
		panic(err)
	}

	require := require.New(t)
	t.Run("basic Containers methods", func(t *testing.T) {
		id, err := containers.ID(testName)
		require.NoError(err)
		require.NotEqual(NullContainerID, id)

		n, err := containers.Container(id)
		require.NoError(err)
		require.Equal(testName, n)

		t.Run("load early stored names", func(t *testing.T) {
			otherVersions := vers.New()
			if err := otherVersions.Prepare(storage); err != nil {
				panic(err)
			}

			otherContainers := New()
			if err := otherContainers.Prepare(storage, versions, nil); err != nil {
				panic(err)
			}

			id1, err := containers.ID(testName)
			require.NoError(err)
			require.Equal(id, id1)

			n1, err := containers.Container(id)
			require.NoError(err)
			require.Equal(testName, n1)
		})
	})
}
