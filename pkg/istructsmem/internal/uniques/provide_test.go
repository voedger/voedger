/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package uniques

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func Test_BasicUsage(t *testing.T) {

	testName := appdef.NewQName("test", "doc")

	testAppDef := func() appdef.IAppDef {
		app := appdef.New()
		def := app.AddStruct(testName, appdef.DefKind_CDoc)
		def.
			AddField("name", appdef.DataKind_string, true).
			AddField("surname", appdef.DataKind_string, false).
			AddField("lastName", appdef.DataKind_string, false).
			AddField("passportNumber", appdef.DataKind_string, false).
			AddField("passportSerial", appdef.DataKind_string, false).
			AddUnique("fullNameUnique", []string{"name", "surname", "lastName"}).
			AddUnique("passportUnique", []string{"passportSerial", "passportNumber"})

		appDef, err := app.Build()
		if err != nil {
			panic(err)
		}

		return appDef
	}

	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	appDef := testAppDef()

	qNames := qnames.New()
	if err := qNames.Prepare(storage, versions, appDef, nil); err != nil {
		panic(err)
	}

	if err := PrepareAppDefUniqueIDs(storage, versions, qNames, appDef); err != nil {
		panic(err)
	}

	require := require.New(t)

	t.Run("test results", func(t *testing.T) {
		def := appDef.Def(testName)

		require.Equal(2, def.UniqueCount())
		require.Equal(def.UniqueCount(), func() int {
			cnt := 0
			def.Uniques(func(u appdef.IUnique) {
				cnt++
				require.Greater(u.ID(), appdef.FirstUniqueID)
			})
			return cnt
		}())
	})
}
