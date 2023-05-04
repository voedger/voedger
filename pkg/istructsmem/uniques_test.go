/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
)

func TestBasicUsage_Uniques(t *testing.T) {
	require := require.New(t)
	test := test()

	qName := appdef.NewQName("my", "name")
	qName2 := appdef.NewQName("my", "name2")
	appDef := appdef.New()

	t.Run("must be ok to build application definition", func(t *testing.T) {
		appDef.AddStruct(qName, appdef.DefKind_CDoc).
			AddField("a", appdef.DataKind_int32, true).
			AddField("b", appdef.DataKind_int32, true).
			AddField("c", appdef.DataKind_int32, true)
	})

	configs := AppConfigsType{}
	cfg := configs.AddConfig(test.appName, appDef)

	// add Uniques in AppConfigType
	cfg.Uniques.Add(qName, []string{"a"})
	cfg.Uniques.Add(qName, []string{"b", "c"})

	// use Uniques using IAppStructs
	asp := Provide(configs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())
	as, err := asp.AppStructs(test.appName)
	require.NoError(err)
	iu := as.Uniques()

	t.Run("GetAll", func(t *testing.T) {
		uniques := iu.GetAll(qName)
		require.Equal([]string{"a"}, uniques[0].Fields())
		require.Equal([]string{"b", "c"}, uniques[1].Fields())
		require.Len(uniques, 2)

		require.Equal(qName, uniques[0].QName())
		require.Equal(qName, uniques[1].QName())

		uniques = iu.GetAll(qName2)
		require.Empty(uniques)
	})
}
