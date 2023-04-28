/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestQNamesBasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	testName := appdef.NewQName("test", "doc")
	app := appdef.New()
	app.Add(testName, appdef.DefKind_CDoc)
	appDef, err := app.Build()
	if err != nil {
		panic(err)
	}

	resources := istructs.IResources(nil) //TODO: add test resources

	names := New()
	if err := names.Prepare(storage, versions, appDef, resources); err != nil {
		panic(err)
	}

	require := require.New(t)
	t.Run("basic QNames methods", func(t *testing.T) {
		id, err := names.GetID(testName)
		require.NoError(err)
		require.NotEqual(NullQNameID, id)

		n, err := names.GetQName(id)
		require.NoError(err)
		require.Equal(testName, n)

		t.Run("must be able to load early stored names", func(t *testing.T) {
			otherVersions := vers.New()
			if err := otherVersions.Prepare(storage); err != nil {
				panic(err)
			}

			otherNames := New()
			if err := otherNames.Prepare(storage, versions, nil, nil); err != nil {
				panic(err)
			}

			id1, err := names.GetID(testName)
			require.NoError(err)
			require.Equal(id, id1)

			n1, err := names.GetQName(id)
			require.NoError(err)
			require.Equal(testName, n1)
		})
	})
}
