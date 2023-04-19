/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_BasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.NewVersions()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	testName := "test"
	bld := schemas.NewSchemaCache()
	bld.Add(istructs.NewQName("test", "schema"), istructs.SchemaKind_Element).
		AddContainer(testName, istructs.NewQName("test", "schema"), 0, istructs.ContainerOccurs_Unbounded)
	schemas, err := bld.Build()
	if err != nil {
		panic(err)
	}

	containers := NewContainers()
	if err := containers.Prepare(storage, versions, schemas); err != nil {
		panic(err)
	}

	require := require.New(t)
	t.Run("basic Containers methods", func(t *testing.T) {
		id, err := containers.GetID(testName)
		require.NoError(err)
		require.NotEqual(NullContainerID, id)

		n, err := containers.GetContainer(id)
		require.NoError(err)
		require.Equal(testName, n)

		t.Run("must be able to load early stored names", func(t *testing.T) {
			otherVersions := vers.NewVersions()
			if err := otherVersions.Prepare(storage); err != nil {
				panic(err)
			}

			otherContainers := NewContainers()
			if err := otherContainers.Prepare(storage, versions, nil); err != nil {
				panic(err)
			}

			id1, err := containers.GetID(testName)
			require.NoError(err)
			require.Equal(id, id1)

			n1, err := containers.GetContainer(id)
			require.NoError(err)
			require.Equal(testName, n1)
		})
	})
}
