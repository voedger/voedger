/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestQNamesBasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(mem.Provide())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	testName := appdef.NewQName("test", "doc")
	adb := appdef.New()
	ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
	doc := ws.AddCDoc(testName)
	doc.AddField("f1", appdef.DataKind_int64, false)
	doc.AddUnique(appdef.UniqueQName(testName, "f1"), []appdef.FieldName{"f1"})
	appDef := adb.MustBuild()

	names := New()
	if err := names.Prepare(storage, versions, appDef); err != nil {
		panic(err)
	}

	require := require.New(t)
	t.Run("basic QNames methods", func(t *testing.T) {
		id, err := names.ID(testName)
		require.NoError(err)
		require.NotEqual(NullQNameID, id)

		n, err := names.QName(id)
		require.NoError(err)
		require.Equal(testName, n)

		t.Run("must be able to load early stored names", func(t *testing.T) {
			otherVersions := vers.New()
			if err := otherVersions.Prepare(storage); err != nil {
				panic(err)
			}

			otherNames := New()
			if err := otherNames.Prepare(storage, versions, nil); err != nil {
				panic(err)
			}

			id1, err := names.ID(testName)
			require.NoError(err)
			require.Equal(id, id1)

			n1, err := names.QName(id)
			require.NoError(err)
			require.Equal(testName, n1)
		})
	})
}
