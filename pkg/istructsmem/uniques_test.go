/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/schemas"
)

func TestBasicUsage_Uniques(t *testing.T) {
	require := require.New(t)
	test := test()

	qName := schemas.NewQName("my", "name")
	qName2 := schemas.NewQName("my", "name2")
	bld := schemas.NewSchemaCache()

	t.Run("must be ok to build schemas", func(t *testing.T) {
		bld.Add(qName, schemas.SchemaKind_CDoc).
			AddField("a", schemas.DataKind_int32, true).
			AddField("b", schemas.DataKind_int32, true).
			AddField("c", schemas.DataKind_int32, true)
	})

	cfgs := AppConfigsType{}
	cfg := cfgs.AddConfig(test.appName, bld)

	// add Uniques in AppConfigType
	cfg.Uniques.Add(qName, []string{"a"})
	cfg.Uniques.Add(qName, []string{"b", "c"})

	// use Uniques using IAppStructs
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
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

	t.Run("GetForKeysSet", func(t *testing.T) {
		u := iu.GetForKeySet(qName, []string{"a"})
		require.Equal([]string{"a"}, u.Fields())

		u = iu.GetForKeySet(qName, []string{"b", "c"})
		require.Equal([]string{"b", "c"}, u.Fields())

		// order has no sense
		u = iu.GetForKeySet(qName, []string{"c", "b"})
		require.Equal([]string{"b", "c"}, u.Fields())

		require.Nil(iu.GetForKeySet(qName, []string{"a", "b"}))
		require.Nil(iu.GetForKeySet(qName, []string{"b"}))
		require.Nil(iu.GetForKeySet(qName, []string{"a", "b", "c"}))
		require.Nil(iu.GetForKeySet(qName2, []string{"any"}))

		require.Panics(func() { iu.GetForKeySet(qName, []string{"b", "b"}) })
		require.Panics(func() { iu.GetForKeySet(qName, []string{"a", "a"}) })
	})
}
