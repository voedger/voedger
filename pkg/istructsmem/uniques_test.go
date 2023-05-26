/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage_Uniques(t *testing.T) {
	require := require.New(t)
	test := test()

	qName := appdef.NewQName("my", "name")
	qName2 := appdef.NewQName("my", "name2")
	appDef := appdef.New()

	t.Run("must be ok to build application definition", func(t *testing.T) {
		appDef.AddCDoc(qName).
			AddField("a", appdef.DataKind_int32, true).
			AddField("b", appdef.DataKind_int32, true).
			AddField("c", appdef.DataKind_int32, true)
	})

	configs := AppConfigsType{}
	cfg := configs.AddConfig(test.appName, appDef)

	// add Uniques in AppConfigType
	cfg.Uniques.Add(qName, []string{"a"})

	// use Uniques using IAppStructs
	asp := Provide(configs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())
	as, err := asp.AppStructs(test.appName)
	require.NoError(err)
	iu := as.Uniques()

	t.Run("GetAll", func(t *testing.T) {
		uniques := iu.GetAll(qName)
		require.Equal([]string{"a"}, uniques[0].Fields())
		require.Len(uniques, 1)

		require.Equal(qName, uniques[0].QName())

		uniques = iu.GetAll(qName2)
		require.Empty(uniques)
	})
}

func TestUniquesRestrictions(t *testing.T) {

	cDoc, obj := appdef.NewQName("test", "cDoc"), appdef.NewQName("test", "obj")

	config := func() *AppConfigType {
		app := appdef.New()
		_ = app.AddCDoc(cDoc).
			AddField("a", appdef.DataKind_int32, true).
			AddField("b", appdef.DataKind_int32, true).
			AddField("c", appdef.DataKind_int32, false)
		_ = app.AddObject(obj).
			AddField("a", appdef.DataKind_int32, true).
			AddField("b", appdef.DataKind_int32, true).
			AddField("c", appdef.DataKind_int32, false)
		configs := AppConfigsType{}
		return configs.AddConfig(istructs.AppQName_test1_app1, app)
	}

	require := require.New(t)

	t.Run("must error if not found definition", func(t *testing.T) {
		cfg := config()
		cfg.Uniques.Add(appdef.NewQName("test", "unknown"), []string{"a"})
		require.ErrorIs(cfg.Uniques.validate(cfg), appdef.ErrNameNotFound)
	})

	t.Run("must error if kind definition unable uniques", func(t *testing.T) {
		cfg := config()
		cfg.Uniques.Add(obj, []string{"a"})
		require.ErrorIs(cfg.Uniques.validate(cfg), appdef.ErrInvalidDefKind)
	})

	t.Run("must error if more then 1 unique per definition", func(t *testing.T) {
		cfg := config()
		cfg.Uniques.Add(cDoc, []string{"a"})
		cfg.Uniques.Add(cDoc, []string{"b"})
		require.ErrorIs(cfg.Uniques.validate(cfg), appdef.ErrTooManyUniques)
	})

	t.Run("must error if more then 1 field per unique", func(t *testing.T) {
		cfg := config()
		cfg.Uniques.Add(cDoc, []string{"a", "b"})
		require.ErrorIs(cfg.Uniques.validate(cfg), appdef.ErrTooManyFields)
	})

	t.Run("must error if field not found", func(t *testing.T) {
		cfg := config()
		cfg.Uniques.Add(cDoc, []string{"unknownField"})
		require.ErrorIs(cfg.Uniques.validate(cfg), appdef.ErrNameNotFound)
	})

	t.Run("must error if field is not required", func(t *testing.T) {
		cfg := config()
		cfg.Uniques.Add(cDoc, []string{"c"})
		require.ErrorIs(cfg.Uniques.validate(cfg), appdef.ErrWrongDefStruct)
	})
}
