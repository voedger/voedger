/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"context"
	"encoding/base64"
	gojson "encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
)

func Test_KeyType(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	viewName := appdef.NewQName("test", "view")

	appConfigs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)

		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		t.Run("must be ok to build view", func(t *testing.T) {
			view := adb.AddView(viewName)
			view.Key().PartKey().
				AddField("pk_int32", appdef.DataKind_int32).
				AddField("pk_int64", appdef.DataKind_int64).
				AddField("pk_float32", appdef.DataKind_float32).
				AddField("pk_float64", appdef.DataKind_float64).
				AddField("pk_qname", appdef.DataKind_QName).
				AddField("pk_bool", appdef.DataKind_bool).
				AddRefField("pk_recID").
				AddField("pk_number", appdef.DataKind_float64)
			view.Key().ClustCols().
				AddField("cc_int32", appdef.DataKind_int32).
				AddField("cc_int64", appdef.DataKind_int64).
				AddField("cc_float32", appdef.DataKind_float32).
				AddField("cc_float64", appdef.DataKind_float64).
				AddField("cc_qname", appdef.DataKind_QName).
				AddField("cc_bool", appdef.DataKind_bool).
				AddRefField("cc_recID").
				AddField("cc_number", appdef.DataKind_float64).
				AddField("cc_bytes", appdef.DataKind_bytes, appdef.MaxLen(64))
			view.Value().
				AddField("val_string", appdef.DataKind_string, false, appdef.MaxLen(1024))
		})

		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		return cfgs
	}

	appCfgs := appConfigs()
	appCfg := appCfgs.GetConfig(appName)

	appProvider := Provide(appCfgs, iratesce.TestBucketsFactory, testTokensFactory(), teststore.NewStorageProvider(teststore.NewStorage(appName)))
	app, err := appProvider.BuiltIn(appName)
	require.NoError(err)
	require.NotNil(app)

	key := newKey(appCfg, viewName)

	t.Run("key must supports IKeyBuilder interface", func(t *testing.T) {
		kb := istructs.IKeyBuilder(key)

		require.NotNil(kb)

		kb.PutInt32("pk_int32", 1111111)
		kb.PutInt64("pk_int64", 222222222222)
		kb.PutFloat32("pk_float32", 3.333e3)
		kb.PutFloat64("pk_float64", -4.4444e-44)
		kb.PutQName("pk_qname", istructs.QNameForError)
		kb.PutBool("pk_bool", true)
		kb.PutRecordID("pk_recID", istructs.RecordID(5555555))
		kb.PutNumber("pk_number", gojson.Number("1.23456789"))

		kb.PutInt32("cc_int32", 6666666)
		kb.PutInt64("cc_int64", 777777777777)
		kb.PutFloat32("cc_float32", 8.888e8)
		kb.PutFloat64("cc_float64", -9.9999e-99)
		kb.PutQName("cc_qname", viewName)
		kb.PutBool("cc_bool", true)
		kb.PutRecordID("cc_recID", istructs.RecordID(314159265358))
		kb.PutNumber("cc_number", gojson.Number("-9.87654321"))
		kb.PutChars("cc_bytes", base64.StdEncoding.EncodeToString([]byte(`naked 🔫`)))
	})

	require.NoError(key.build())

	t.Run("should be ok IKeyBuilder.ToBytes()", func(t *testing.T) {
		pk, cc, err := key.ToBytes(0)
		require.NoError(err)
		require.NotEmpty(pk)
		require.NotEmpty(cc)
	})

	testIKey := func(t *testing.T, key *keyType) {
		k := istructs.IKey(key)

		require.NotNil(k)

		require.EqualValues(1111111, k.AsInt32("pk_int32"))
		require.EqualValues(222222222222, k.AsInt64("pk_int64"))
		require.EqualValues(3.333e3, k.AsFloat32("pk_float32"))
		require.EqualValues(-4.4444e-44, k.AsFloat64("pk_float64"))
		require.EqualValues(istructs.QNameForError, k.AsQName("pk_qname"))
		require.True(k.AsBool("pk_bool"))
		require.EqualValues(5555555, k.AsRecordID("pk_recID"))
		require.EqualValues(1.23456789, k.AsFloat64("pk_number"))

		require.EqualValues(6666666, k.AsInt32("cc_int32"))
		require.EqualValues(777777777777, k.AsInt64("cc_int64"))
		require.EqualValues(8.888e8, k.AsFloat32("cc_float32"))
		require.EqualValues(-9.9999e-99, k.AsFloat64("cc_float64"))
		require.EqualValues(viewName, k.AsQName("cc_qname"))
		require.True(k.AsBool("cc_bool"))
		require.EqualValues(314159265358, k.AsRecordID("cc_recID"))
		require.EqualValues(-9.87654321, k.AsFloat64("cc_number"))
		require.EqualValues(`naked 🔫`, k.AsBytes("cc_bytes"))

		t.Run("should be ok to enum IKey.FieldNames", func(t *testing.T) {
			view := appCfg.AppDef.View(viewName)
			cnt := 0
			for n := range k.FieldNames {
				require.NotNil(view.Key().Field(n), "unknown field name passed in callback from IKey.FieldNames(): %q", n)
				cnt++
			}
			require.Positive(cnt)
			require.Equal(view.Key().FieldCount(), cnt)
		})

		t.Run("should be ok to enum all ids with IKey.RecordIDs()", func(t *testing.T) {
			cnt := 0
			for n, id := range k.RecordIDs(true) {
				switch n {
				case "pk_recID":
					require.EqualValues(5555555, id)
				case "cc_recID":
					require.EqualValues(314159265358, id)
				default:
					require.Fail("unexpected field in range IKey.RecordIDs()", "fieldName: %q", n)
				}
				cnt++
			}
			require.Equal(2, cnt)
		})
	}

	t.Run("key must supports IKey interface", func(t *testing.T) { testIKey(t, key) })

	t.Run("must be ok to load/store key to bytes", func(t *testing.T) {
		p, c := key.storeToBytes(0)
		require.NotEmpty(p)
		require.NotEmpty(c)

		dupe := newKey(appCfg, viewName)
		dupe.partRow.copyFrom(&key.partRow)
		require.NoError(dupe.loadFromBytes(c))

		t.Run("key must supports IKey interface", func(t *testing.T) { testIKey(t, dupe) })

		require.True(key.Equals(dupe))
	})

	t.Run("should be ok IValueBuilder.ToBytes()", func(t *testing.T) {
		vb := newValue(appCfg, viewName)
		vb.PutString("val_string", "test string")

		b, err := vb.ToBytes()
		require.NoError(err)
		require.NotEmpty(b)
	})
}

// TestCore_ViewRecords: test https://dev.heeus.io/launchpad/#!14470
func TestCore_ViewRecords(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	storage := teststore.NewStorage(appName)
	storageProvider := teststore.NewStorageProvider(storage)

	appConfigs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)

		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		t.Run("must be ok to build application", func(t *testing.T) {
			view := adb.AddView(appdef.NewQName("test", "viewDrinks"))
			view.Key().PartKey().
				AddField("partitionKey1", appdef.DataKind_int64)
			view.Key().ClustCols().
				AddField("clusteringColumn1", appdef.DataKind_int64).
				AddField("clusteringColumn2", appdef.DataKind_bool).
				AddField("clusteringColumn3", appdef.DataKind_string, appdef.MaxLen(64))
			view.Value().
				AddField("id", appdef.DataKind_int64, true).
				AddField("name", appdef.DataKind_string, true).
				AddField("active", appdef.DataKind_bool, true)

			otherView := adb.AddView(appdef.NewQName("test", "otherView"))
			otherView.Key().PartKey().
				AddField("partitionKey1", appdef.DataKind_QName)
			otherView.Key().ClustCols().
				AddField("clusteringColumn1", appdef.DataKind_float32).
				AddField("clusteringColumn2", appdef.DataKind_float64).
				AddField("clusteringColumn3", appdef.DataKind_bytes, appdef.MaxLen(128))
			otherView.Value().
				AddField("valueField1", appdef.DataKind_int64, false)
		})

		cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		return cfgs
	}

	appCfgs := appConfigs()
	appCfg := appCfgs.GetConfig(istructs.AppQName_test1_app1)
	p := Provide(appCfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
	app, err := p.BuiltIn(istructs.AppQName_test1_app1)
	require.NoError(err)
	viewRecords := app.ViewRecords()

	t.Run("must be ok put records one-by-one", func(t *testing.T) {
		entries := []entryType{
			newEntry(viewRecords, 1, 100, true, "soda", 1, "Cola"),
			newEntry(viewRecords, 1, 100, true, "soda", 2, "Cola light"), // dupe, must override previous name
			newEntry(viewRecords, 2, 100, true, "soda", 2, "Pepsi"),
			newEntry(viewRecords, 2, 100, true, "cider", 2, "Apple cider"),
		}
		for _, e := range entries {
			err := viewRecords.Put(e.wsid, e.key, e.value)
			require.NoError(err)
		}
	})

	t.Run("Should read one (!) record by WSID = 1", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 1)

		var (
			fk_part  int64
			fk_c1    int64
			fk_c2    bool
			fk_c3    string
			val_name string
		)
		counter := 0
		err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			counter++
			fk_part = key.AsInt64("partitionKey1")
			fk_c1 = key.AsInt64("clusteringColumn1")
			fk_c2 = key.AsBool("clusteringColumn2")
			fk_c3 = key.AsString("clusteringColumn3")
			val_name = value.AsString("name")
			return nil
		})
		require.NoError(err)

		require.Equal(1, counter)

		require.Equal(int64(1), fk_part)

		require.Equal(int64(100), fk_c1)
		require.True(fk_c2)
		require.Equal("soda", fk_c3)

		require.Equal("Cola light", val_name)
	})

	t.Run("must be ok batch put", func(t *testing.T) {
		entries := []entryType{
			newEntry(viewRecords, 3, 200, true, "food", 1, "Meat"),
			newEntry(viewRecords, 3, 300, true, "food", 1, "Bread"),
			newEntry(viewRecords, 3, 400, true, "food", 1, "Cake"),
		}

		batch := make([]istructs.ViewKV, len(entries))
		for i, e := range entries {
			batch[i].Key = e.key
			batch[i].Value = e.value
		}
		err := viewRecords.PutBatch(3, batch)
		require.NoError(err)
	})

	t.Run("Should read three record from WSID = 3 with correct order", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 3)

		counter, names := 0, ""
		err := viewRecords.Read(context.Background(), 3, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			counter++
			names += value.AsString("name") + ";"
			return nil
		})
		require.NoError(err)

		require.Equal(3, counter)
		require.Equal("Meat;Bread;Cake;", names, "wrong read order!")
	})

	t.Run("Should read two records by short clustering key and one by full", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)

		t.Run("Should read one records by short clustering key", func(t *testing.T) {
			counter, val_names := 0, "|"
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_names += value.AsString("name") + "|"
				return nil
			})
			require.NoError(err)
			require.Equal(2, counter)
			require.Equal("|Apple cider|Pepsi|", val_names)
		})

		t.Run("Should read one records by short «c» clustering key", func(t *testing.T) {
			kb.PutString("clusteringColumn3", "c")
			counter, val_name := 0, "|"
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_name += value.AsString("name") + "|"
				return nil
			})
			require.NoError(err)
			require.Equal(1, counter)
			require.Equal("|Apple cider|", val_name)
		})

		t.Run("Should read one record by long «cid» clustering key", func(t *testing.T) {
			kb.PutString("clusteringColumn3", "cid")
			counter, val_name := 0, "|"
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_name += value.AsString("name") + "|"
				return nil
			})
			require.NoError(err)
			require.Equal(1, counter)
			require.Equal("|Apple cider|", val_name)
		})

		t.Run("Should no read records by not existing clustering key", func(t *testing.T) {
			kb.PutString("clusteringColumn3", "tofu")
			counter := 0
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				return nil
			})
			require.NoError(err)
			require.Equal(0, counter)
		})

		t.Run("Should read one records by short «s» clustering key. Old style key filling", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PartitionKey().PutInt64("partitionKey1", 2)
			kb.ClusteringColumns().PutInt64("clusteringColumn1", 100)
			kb.ClusteringColumns().PutBool("clusteringColumn2", true)
			kb.ClusteringColumns().PutString("clusteringColumn3", "s")
			counter := 0
			val_name := "|"
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_name += value.AsString("name") + "|"
				return nil
			})
			require.NoError(err)
			require.Equal(1, counter)
			require.Equal("|Pepsi|", val_name)
		})
	})

	t.Run("get exists record must be ok", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "cider")

		value, err := viewRecords.Get(2, kb)
		require.NoError(err)
		require.Equal(int64(2), value.AsInt64("id"))
		require.Equal("Apple cider", value.AsString("name"))
		require.True(value.AsBool("active"))
	})

	t.Run("get not exists record must be available", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "tofu")

		value, err := viewRecords.Get(2, kb)
		require.ErrorIs(err, ErrRecordNotFound)
		require.NotNil(value)
	})

	t.Run("Test UpdateValueBuilder", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 1)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "soda")

		oldValue := istructs.IValue(nil)
		entryName := ""

		err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			oldValue = value
			entryName = value.AsString("name")
			return nil
		})

		require.NoError(err)
		require.NotNil(oldValue)
		require.Equal("Cola light", entryName)

		vb := viewRecords.UpdateValueBuilder(appdef.NewQName("test", "viewDrinks"), oldValue)
		vb.PutString("name", "Cola lemon")

		err = viewRecords.Put(1, kb, vb)
		require.NoError(err)

		err = viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			entryName = value.AsString("name")
			return nil
		})

		require.NoError(err)
		require.Equal("Cola lemon", entryName)
	})

	t.Run("Invalid key building test", func(t *testing.T) {

		require.Panics(func() { _ = viewRecords.KeyBuilder(appdef.NullQName) },
			require.Is(ErrNameMissed, "Should panics if key type missed"))

		require.Panics(func() { _ = viewRecords.KeyBuilder(istructs.QNameForError) },
			require.Is(ErrNameNotFound), require.Has(istructs.QNameForError))
		require.Panics(func() { _ = viewRecords.KeyBuilder(appdef.NewQName("test", "unknownDrinks")) },
			require.Is(ErrNameNotFound), require.Has("test.unknownDrinks"))

		require.Panics(func() { _ = viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks_Value")) },
			require.Is(ErrNameNotFound), require.Has("test.viewDrinks_Value"))

		t.Run("Must have error if wrong partition key type", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			pk := kb.PartitionKey()
			pk.PutQName(appdef.SystemField_QName, appdef.NewQName("test", "viewDrinks_Value"))
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrTypeChanged)
		})

		t.Run("Must have error if wrong clustering columns type", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			pk := kb.PartitionKey()
			pk.PutInt64("partitionKey1", 1)
			cc := kb.ClusteringColumns()
			cc.PutQName(appdef.SystemField_QName, appdef.NewQName("test", "viewDrinks_Value"))
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrTypeChanged)
		})

		t.Run("Must have error if holes in clustering column", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			pk := kb.PartitionKey()
			pk.PutInt64("partitionKey1", 1)
			cc := kb.ClusteringColumns()
			cc.PutInt64("clusteringColumn1", 100)
			cc.PutString("clusteringColumn3", "s")
			cnt := 0
			err := viewRecords.Read(context.Background(), 1, kb, func(istructs.IKey, istructs.IValue) (err error) {
				cnt++
				return nil
			})
			require.ErrorIs(err, ErrFieldIsEmpty)
			require.ErrorContains(err, "hole at field «clusteringColumn2»")
			require.Zero(cnt)
		})

		t.Run("Must have error if wrong value type", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.PutInt64("clusteringColumn1", 100)
			kb.PutBool("clusteringColumn2", true)
			kb.PutString("clusteringColumn3", "soda")

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
			vb.PutQName(appdef.SystemField_QName, appdef.NewQName("test", "viewDrinks_ClusteringColumns"))

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrTypeChanged)
		})

		t.Run("Must have error if empty partition key", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))

			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrFieldIsEmpty)

			validateErr := validateErrorf(0, "")
			require.ErrorAs(err, &validateErr)
			require.Equal(ECode_EmptyData, validateErr.Code())

			_, err = viewRecords.Get(1, kb)
			require.ErrorIs(err, ErrFieldIsEmpty)
		})

		t.Run("Must have error for put if empty clustering columns key", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
			vb.PutInt64("id", 1)
			vb.PutString("name", "tea")
			vb.PutBool("active", true)

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrFieldIsEmpty)

			validateErr := validateErrorf(0, "")
			require.ErrorAs(err, &validateErr)
			require.Equal(ECode_EmptyData, validateErr.Code())
		})

		t.Run("Must have error if wrong fields in key", func(t *testing.T) {

			t.Run("Must put error", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
				kb.PutBool("errorField", true)
				err := viewRecords.Put(1, kb, nil)
				require.ErrorIs(err, ErrNameNotFound)

				t.Run("should be error IKeyBuilder.ToBytes()", func(t *testing.T) {
					pk, cc, err := kb.ToBytes(0)
					require.ErrorIs(err, ErrNameNotFound)
					require.Empty(pk)
					require.Empty(cc)
				})
			})

			t.Run("Must read and error", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
				kb.PutBool("errorField", true)
				err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
					return nil
				})
				require.ErrorIs(err, ErrNameNotFound)

				_, err = viewRecords.Get(1, kb)
				require.ErrorIs(err, ErrNameNotFound)
			})

		})

		t.Run("Must have error if wrong fields in partition key", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PartitionKey().PutBool("errorField", true)
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrNameNotFound)

			_, err = viewRecords.Get(1, kb)
			require.ErrorIs(err, ErrNameNotFound)
		})

		t.Run("Must have error if wrong fields in clustering columns", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.ClusteringColumns().PutBytes("errorField", []byte{1, 2, 3})
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrNameNotFound)

			_, err = viewRecords.Get(1, kb)
			require.ErrorIs(err, ErrNameNotFound)
		})
	})

	t.Run("Invalid value building test", func(t *testing.T) {

		require.Panics(func() { _ = viewRecords.NewValueBuilder(appdef.NullQName) },
			require.Is(ErrNameMissed, "Should panics if value type missed"))

		require.Panics(func() { _ = viewRecords.NewValueBuilder(appdef.NewQName("test", "unknownDrinks")) },
			require.Is(ErrNameNotFound), require.Has("test.unknownDrinks"))

		require.Panics(func() { _ = viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks_PartitionKey")) },
			require.Is(ErrNameNotFound), require.Has("test.viewDrinks_PartitionKey"))

		t.Run("Must have panic if wrong existing value type specified", func(t *testing.T) {
			exists := newValue(appCfg, appdef.NewQName("test", "otherView"))
			require.Panics(func() {
				_ = viewRecords.UpdateValueBuilder(appdef.NewQName("test", "viewDrinks"), exists)
			}, require.Is(ErrWrongType), require.Has("test.otherView"))
		})

		t.Run("Must have error for put if empty value", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.PutInt64("clusteringColumn1", 100)
			kb.PutBool("clusteringColumn2", true)
			kb.PutString("clusteringColumn3", "null")

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrNameNotFound)

			validateErr := validateErrorf(0, "")
			require.ErrorAs(err, &validateErr)
			require.Equal(ECode_EmptyData, validateErr.Code())
		})

		t.Run("Must have error if errors in value", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "otherView"))
			kb.PutQName("partitionKey1", appdef.NullQName)
			kb.PutFloat32("clusteringColumn1", 44.4)
			kb.PutFloat64("clusteringColumn2", 64.4)
			kb.PutBytes("clusteringColumn3", []byte("TEST"))

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "otherView"))
			vb.PutQName("unknownField", appdef.NullQName)

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrNameNotFound)

			t.Run("should be error IValueBuilder.ToBytes()", func(t *testing.T) {
				v, err := vb.ToBytes()
				require.ErrorIs(err, ErrNameNotFound)
				require.Empty(v)
			})
		})

		t.Run("Must have error if key and value are from different views", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "otherView"))
			kb.PutQName("partitionKey1", appdef.NullQName)
			kb.PutFloat32("clusteringColumn1", 44.4)
			kb.PutFloat64("clusteringColumn2", 64.4)
			kb.PutBytes("clusteringColumn3", []byte("TEST"))

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
			vb.PutInt64("id", 1)
			vb.PutString("name", "baikal")
			vb.PutBool("active", true)

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrWrongType)
			require.ErrorContains(err, "test.viewDrinks")
		})

		t.Run("put batch must fail if error in any key-value item", func(t *testing.T) {
			entries := []entryType{
				newEntry(viewRecords, 7, 200, true, "food", 1, "Meat"),
				newEntry(viewRecords, 7, 300, true, "food", 1, "Bread"),
				newEntry(viewRecords, 7, 400, true, "food", 1, "Cake"),
			}

			entries[1].value.PutBool("errorField", true)

			batch := make([]istructs.ViewKV, len(entries))
			for i, e := range entries {
				batch[i].Key = e.key
				batch[i].Value = e.value
			}
			err := viewRecords.PutBatch(7, batch)
			require.ErrorIs(err, ErrNameNotFound)

			t.Run("put batch failed; no record from WSID = 7 must be read", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
				kb.PutInt64("partitionKey1", 7)

				require.NoError(viewRecords.Read(context.Background(), 7, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
					require.Fail("if put batch failed then no records must be read")
					return nil
				}))
			})
		})

		t.Run("Must have not error if all is ok", func(t *testing.T) {

			t.Run("basic case", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "otherView"))
				kb.PutQName("partitionKey1", istructs.QNameForError)
				kb.PutFloat32("clusteringColumn1", 44.4)
				kb.PutFloat64("clusteringColumn2", 64.4)
				kb.PutBytes("clusteringColumn3", []byte("TEST"))

				vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "otherView"))
				vb.PutInt64("valueField1", 1)

				err := viewRecords.Put(1, kb, vb)
				require.NoError(err)

				counter := 0
				err = viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
					require.Equal(istructs.QNameForError, key.AsQName("partitionKey1"))
					require.Equal(float32(44.4), key.AsFloat32("clusteringColumn1"))
					require.Equal(float64(64.4), key.AsFloat64("clusteringColumn2"))
					require.Equal([]byte("TEST"), key.AsBytes("clusteringColumn3"))
					require.Equal(int64(1), value.AsInt64("valueField1"))
					counter++
					return nil
				})
				require.NoError(err)
				require.Equal(1, counter)
			})

			t.Run("full bytes clustering columns case", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "otherView"))
				kb.PutQName("partitionKey1", istructs.QNameForError)
				kb.PutFloat32("clusteringColumn1", 44.4)
				kb.PutFloat64("clusteringColumn2", 64.4)
				kb.PutBytes("clusteringColumn3", []byte{0xFF, 0x1, 0x2})

				vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "otherView"))
				vb.PutInt64("valueField1", 7)

				err := viewRecords.Put(1, kb, vb)
				require.NoError(err)

				readKey := viewRecords.KeyBuilder(appdef.NewQName("test", "otherView"))
				readKey.PutQName("partitionKey1", istructs.QNameForError)
				readKey.PutFloat32("clusteringColumn1", 44.4)
				readKey.PutFloat64("clusteringColumn2", 64.4)
				readKey.PutBytes("clusteringColumn3", []byte{0xFF})

				counter := 0
				err = viewRecords.Read(context.Background(), 1, readKey, func(key istructs.IKey, value istructs.IValue) (err error) {
					require.Equal(istructs.QNameForError, key.AsQName("partitionKey1"))
					require.Equal(float32(44.4), key.AsFloat32("clusteringColumn1"))
					require.Equal(float64(64.4), key.AsFloat64("clusteringColumn2"))
					require.Equal([]byte{0xFF, 0x1, 0x2}, key.AsBytes("clusteringColumn3"))
					require.Equal(int64(7), value.AsInt64("valueField1"))
					counter++
					return nil
				})
				require.NoError(err)
				require.Equal(1, counter)
			})
		})
	})

	t.Run("must Get() and Read() fails if storage returns damaged data", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "cider")

		_, c := kb.(*keyType).storeToBytes(2)

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, c)
		_, err := viewRecords.Get(2, kb)
		require.ErrorIs(err, ErrUnknownCodec)

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, c)
		err = viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) { return nil })
		require.ErrorIs(err, ErrUnknownCodec)
	})
	t.Run("Value builder must build value", func(t *testing.T) {
		vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
		vb.PutInt64("id", 42)
		vb.PutString("name", "Coca Cola")
		vb.PutBool("active", true)

		v := vb.Build()

		require.Equal(int64(42), v.AsInt64("id"))
		require.Equal("Coca Cola", v.AsString("name"))
		require.True(v.AsBool("active"))
	})
	t.Run("Value builder must panic on build value", func(t *testing.T) {
		vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
		vb.PutInt32("id", 42)

		require.Panics(func() { _ = vb.Build() }, require.Is(ErrWrongFieldType), require.Has("id"))
	})
}

type entryType struct {
	wsid  istructs.WSID
	key   istructs.IKeyBuilder
	value istructs.IValueBuilder
}

func newEntry(viewRecords istructs.IViewRecords, wsid istructs.WSID, idDepartment int64, active bool, code string, id int64, name string) entryType {
	kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
	kb.PutInt64("partitionKey1", int64(wsid))
	kb.PutInt64("clusteringColumn1", idDepartment)
	kb.PutBool("clusteringColumn2", active)
	kb.PutString("clusteringColumn3", code)
	vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
	vb.PutInt64("id", id)
	vb.PutString("name", name)
	vb.PutBool("active", active)
	return entryType{
		wsid:  wsid,
		key:   kb,
		value: vb,
	}
}

func Test_ViewRecordsPutJSON(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	const viewName = `test.view`

	storage := teststore.NewStorage(appName)
	storageProvider := teststore.NewStorageProvider(storage)

	appCfgs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)

		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		t.Run("must be ok to build application", func(t *testing.T) {
			view := adb.AddView(appdef.MustParseQName(viewName))
			view.Key().PartKey().
				AddField("pk1", appdef.DataKind_int64)
			view.Key().ClustCols().
				AddField("cc1", appdef.DataKind_int64).
				AddField("cc2", appdef.DataKind_string, appdef.MaxLen(64))
			view.Value().
				AddField("v1", appdef.DataKind_float32, true).
				AddField("v2", appdef.DataKind_string, true)
		})
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		return cfgs
	}()

	app, err := Provide(appCfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider).BuiltIn(appName)
	require.NoError(err)

	t.Run("should be ok to put view record via PutJSON", func(t *testing.T) {
		json := make(map[appdef.FieldName]any)
		json[appdef.SystemField_QName] = viewName
		json["pk1"] = gojson.Number("1")
		json["cc1"] = gojson.Number("2")
		json["cc2"] = "test sort"
		json["v1"] = gojson.Number("3")
		json["v2"] = "naked 🔫"

		err := app.ViewRecords().PutJSON(33, json)
		require.NoError(err)

		t.Run("should be ok to read view record", func(t *testing.T) {
			k := app.ViewRecords().KeyBuilder(appdef.MustParseQName(viewName))
			k.PutInt64("pk1", 1)
			k.PutInt64("cc1", 2)
			k.PutString("cc2", "test sort")
			v, err := app.ViewRecords().Get(33, k)
			require.NoError(err)

			require.EqualValues(viewName, v.AsQName(appdef.SystemField_QName).String())
			require.EqualValues(3, v.AsFloat32("v1"))
			require.EqualValues("naked 🔫", v.AsString("v2"))
		})
	})

	t.Run("errors test", func(t *testing.T) {
		var err error
		t.Run("should be error if wrong view name", func(t *testing.T) {
			json := make(map[appdef.FieldName]any)

			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrFieldIsEmpty)
			require.ErrorContains(err, appdef.SystemField_QName)

			json[appdef.SystemField_QName] = appdef.NullQName.String()
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrFieldIsEmpty)
			require.ErrorContains(err, appdef.SystemField_QName)

			json[appdef.SystemField_QName] = 123
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrWrongFieldType)
			require.ErrorContains(err, appdef.SystemField_QName)

			json[appdef.SystemField_QName] = `naked 🔫`
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, appdef.ErrConvertError)
			require.ErrorContains(err, appdef.SystemField_QName)

			json[appdef.SystemField_QName] = `test.unknown`
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrNameNotFound)
			require.ErrorContains(err, `test.unknown`)
		})

		t.Run("should be error if key errors", func(t *testing.T) {
			json := make(map[appdef.FieldName]any)
			json[appdef.SystemField_QName] = viewName

			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrFieldIsEmpty)
			require.ErrorContains(err, "pk1")

			json["pk1"] = "error value"
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrWrongFieldType)
			require.ErrorContains(err, "pk1")

			json["pk1"] = gojson.Number("1")
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrFieldIsEmpty)
			require.ErrorContains(err, "cc1")

			json["pk1"] = gojson.Number("1")
			json["cc1"] = gojson.Number("2")
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrFieldIsEmpty)
			require.ErrorContains(err, "cc2")
		})

		t.Run("should be error if value errors", func(t *testing.T) {
			json := make(map[appdef.FieldName]any)
			json[appdef.SystemField_QName] = viewName
			json["pk1"] = gojson.Number("1")
			json["cc1"] = gojson.Number("2")
			json["cc2"] = `test sort`

			json["unknownField"] = `value`
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrNameNotFound)
			require.ErrorContains(err, "unknownField")

			delete(json, "unknownField")
			json["v1"] = `value`
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, ErrWrongFieldType)
			require.ErrorContains(err, "v1")
		})
	})
}

func Test_LoadStoreViewRecord_Bytes(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	viewName := appdef.NewQName("test", "view")

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")
	t.Run("must be ok to build application", func(t *testing.T) {
		v := adb.AddView(viewName)
		v.Key().PartKey().
			AddField("pf_int32", appdef.DataKind_int32).
			AddField("pf_int64", appdef.DataKind_int64).
			AddField("pf_float32", appdef.DataKind_float32).
			AddField("pf_float64", appdef.DataKind_float64).
			AddField("pf_qname", appdef.DataKind_QName).
			AddField("pf_bool", appdef.DataKind_bool).
			AddRefField("pf_recID")
		v.Key().ClustCols().
			AddField("cc_int32", appdef.DataKind_int32).
			AddField("cc_int64", appdef.DataKind_int64).
			AddField("cc_float32", appdef.DataKind_float32).
			AddField("cc_float64", appdef.DataKind_float64).
			AddField("cc_qname", appdef.DataKind_QName).
			AddField("cc_bool", appdef.DataKind_bool).
			AddRefField("cc_recID").
			AddField("cc_bytes", appdef.DataKind_bytes, appdef.MaxLen(8))
		v.Value().
			AddField("vf_int32", appdef.DataKind_int32, true).
			AddField("vf_int64", appdef.DataKind_int64, false).
			AddField("vf_float32", appdef.DataKind_float32, false).
			AddField("vf_float64", appdef.DataKind_float64, false).
			AddField("vf_bytes", appdef.DataKind_bytes, false, appdef.MaxLen(1024)).
			AddField("vf_string", appdef.DataKind_string, false, appdef.Pattern(`^\w+$`)).
			AddField("vf_qname", appdef.DataKind_QName, false).
			AddField("vf_bool", appdef.DataKind_bool, false).
			AddRefField("vf_recID", false).
			AddField("vf_record", appdef.DataKind_Record, false).
			AddField("vf_event", appdef.DataKind_Event, false)
	})

	cfg := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app2, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		asp := simpleStorageProvider()
		storage, err := asp.AppStorage(appName)
		require.NoError(err)
		err = cfg.prepare(nil, storage)
		if err != nil {
			panic(err)
		}

		return cfg
	}()

	k1 := newKey(cfg, viewName)
	k1.PutInt32("pf_int32", 1)
	k1.PutInt64("pf_int64", 2)
	k1.PutFloat32("pf_float32", 3.3)
	k1.PutFloat64("pf_float64", 4.4)
	k1.PutQName("pf_qname", istructs.QNameForError)
	k1.PutBool("pf_bool", true)
	k1.PutRecordID("pf_recID", istructs.RecordID(100500))
	k1.PutInt32("cc_int32", 5)
	k1.PutInt64("cc_int64", 6)
	k1.PutFloat32("cc_float32", 7.7)
	k1.PutFloat64("cc_float64", 8.8)
	k1.PutQName("cc_qname", viewName)
	k1.PutBool("cc_bool", true)
	k1.PutRecordID("cc_recID", istructs.RecordID(101501))
	k1.PutBytes("cc_bytes", []byte("test"))
	err := k1.build()
	require.NoError(err)

	p, c := k1.storeToBytes(0)
	require.NotNil(p)
	require.NotNil(c)

	t.Run("should be success load", func(t *testing.T) {
		k2 := newKey(cfg, viewName)
		k2.partRow.copyFrom(&k1.partRow)
		err := k2.loadFromBytes(c)
		require.NoError(err)

		testRowsIsEqual(t, &k1.partRow, &k2.partRow)
		testRowsIsEqual(t, &k1.ccolsRow, &k2.ccolsRow)

		require.True(k1.Equals(k2))
		require.True(k2.Equals(k1))

		k2.PutBytes("cc_bytes", []byte("TesT"))
		require.False(k1.Equals(k2))
	})

	t.Run("should be load error if truncated clustering columns bytes", func(t *testing.T) {
		k2 := newKey(cfg, viewName)
		for i := 0; i < len(c)-4; i++ { // 4 - is length of variable bytes "test" that can be truncated with impunity
			err := k2.loadFromBytes(c[:i])
			require.Error(err, i)
		}
	})

	v1 := newValue(cfg, viewName)
	v1.PutInt32("vf_int32", 1)
	v1.PutInt64("vf_int64", 2)
	v1.PutFloat32("vf_float32", 3.3)
	v1.PutFloat64("vf_float64", 4.4)
	v1.PutBytes("vf_bytes", []byte("test"))
	v1.PutString("vf_string", "test")
	v1.PutQName("vf_qname", viewName)
	v1.PutBool("vf_bool", true)
	v1.PutRecordID("vf_recID", istructs.RecordID(102502))
	v1.PutRecord("vf_record", NewNullRecord(istructs.NullRecordID))
	require.NoError(v1.build())

	v := v1.storeToBytes()

	v2 := newValue(cfg, viewName)
	err = v2.loadFromBytes(v)
	require.NoError(err)

	testRowsIsEqual(t, &v1.rowType, &v2.rowType)

	t.Run("should be load error if truncated value bytes", func(t *testing.T) {
		for i := 0; i < len(v); i++ {
			v2 := newValue(cfg, viewName)
			err := v2.loadFromBytes(v[:i])
			require.Error(err, i)
		}
	})
}

// Test_ViewRecords_ClustColumnsQName: see https://dev.heeus.io/launchpad/#!16377 problem
func Test_ViewRecords_ClustColumnsQName(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	ws := istructs.WSID(1234)

	// Application, same as previous but with RecordID field in the clustering key
	//
	appConfigs := func() AppConfigsType {

		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		t.Run("must be ok to build application", func(t *testing.T) {
			v := adb.AddView(appdef.NewQName("test", "viewDrinks"))
			v.Key().PartKey().
				AddField("partitionKey1", appdef.DataKind_int64)
			v.Key().ClustCols().
				AddField("clusteringColumn1", appdef.DataKind_QName).
				AddRefField("clusteringColumn2")
			v.Value().
				AddField("id", appdef.DataKind_int64, true).
				AddField("name", appdef.DataKind_string, true).
				AddField("active", appdef.DataKind_bool, true)

			_ = adb.AddObject(appdef.NewQName("test", "obj1"))
		})

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		return cfgs
	}

	p := Provide(appConfigs(), iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())
	as, err := p.BuiltIn(appName)
	require.NoError(err)
	viewRecords := as.ViewRecords()

	//
	// Add single record
	//
	kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
	kb.PutInt64("partitionKey1", int64(1))
	kb.PutQName("clusteringColumn1", appdef.NewQName("test", "obj1"))
	kb.PutRecordID("clusteringColumn2", 131072)
	vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
	vb.PutInt64("id", 123)
	vb.PutString("name", "Coca-cola")
	vb.PutBool("active", true)

	require.NoError(viewRecords.Put(ws, kb, vb))

	//
	// Fetch single record
	//
	t.Run("Test read single item", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", int64(1))
		kb.PutQName("clusteringColumn1", appdef.NewQName("test", "obj1"))
		kb.PutRecordID("clusteringColumn2", 131072)

		oldValue := istructs.IValue(nil)
		oldCcKey1 := appdef.NullQName
		oldCcKey2 := istructs.NullRecordID

		err := viewRecords.Read(context.Background(), ws, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			oldCcKey1 = key.AsQName("clusteringColumn1")
			oldCcKey2 = key.AsRecordID("clusteringColumn2")
			oldValue = value
			return nil
		})
		require.NoError(err)
		require.NotNil(oldValue)
		require.Equal("Coca-cola", oldValue.AsString("name"))
		require.Equal(appdef.NewQName("test", "obj1"), oldCcKey1)
		require.Equal(istructs.RecordID(131072), oldCcKey2)
	})
}

func Test_ViewRecord_GetBatch(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	championshipsView := appdef.NewQName("test", "championships")
	championsView := appdef.NewQName("test", "champions")

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")
	t.Run("must be ok to build application", func(t *testing.T) {
		v := adb.AddView(championshipsView)
		v.Key().PartKey().
			AddField("Year", appdef.DataKind_int32)
		v.Key().ClustCols().
			AddField("Sport", appdef.DataKind_string, appdef.MaxLen(64))
		v.Value().
			AddField("Country", appdef.DataKind_string, true).
			AddField("City", appdef.DataKind_string, false)

		v = adb.AddView(championsView)
		v.Key().PartKey().
			AddField("Year", appdef.DataKind_int32)
		v.Key().ClustCols().
			AddField("Sport", appdef.DataKind_string, appdef.MaxLen(64))
		v.Value().
			AddField("Winner", appdef.DataKind_string, true)
	})

	storage := teststore.NewStorage(appName)
	storageProvider := teststore.NewStorageProvider(storage)

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	type championship struct {
		year                 int32
		sport, country, city string
		winner               string
	}
	var championships = []championship{
		{1949, "Волейбол", "Чехословакия", "Прага", "СССР"},
		{1952, "Волейбол", "СССР", "Москва", "СССР"},
		{1956, "Волейбол", "Франция", "Париж", "Чехословакия"},
		{1960, "Волейбол", "Бразилия", "Рио-де-Жанейро", "СССР"},
		{1962, "Волейбол", "СССР", "Москва", "СССР"},
		{1966, "Волейбол", "Чехословакия", "Прага", "Чехословакия"},
		{1970, "Волейбол", "Болгария", "София", "ГДР"},
		{1974, "Волейбол", "Мексика", "Мехико", "Польша"},
		{1978, "Волейбол", "Италия", "Рим", "СССР"},
		{1982, "Волейбол", "Аргентина", "Буэнос-Айрес", "СССР"},
		{1986, "Волейбол", "Франция", "Париж", "США"},
		{1990, "Волейбол", "Бразилия", "Рио-де-Жанейро", "Италия"},
		{1994, "Волейбол", "Греция", "Афины", "Италия"},
		{1998, "Волейбол", "Япония", "Токио", "Италия"},
		{2002, "Волейбол", "Аргентина", "Буэнос-Айрес", "Бразилия"},
		{2006, "Волейбол", "Япония", "Токио", "Бразилия"},
		{2010, "Волейбол", "Италия", "Рим", "Бразилия"},
		{2014, "Волейбол", "Польша", "Катовице", "Польша"},
		{2018, "Волейбол", "Италия", "Рим", "Польша"},
		{2022, "Волейбол", "Россия", "Москва", ""},

		{1938, "Гандбол", "Германия", "Берлин", "Германия"},
		{1942, "Гандбол", "Швеция", "Осло", "Швеция"},
		{1958, "Гандбол", "ГДР", "Берлин", "Швеция"},
		{1961, "Гандбол", "ФРГ", "Бонн", "Румыния"},
		{1964, "Гандбол", "Чехословакия", "Прага", "Румыния"},
		{1967, "Гандбол", "Швеция", "Осло", "Чехословакия"},
		{1970, "Гандбол", "Франция", "Париж", "Румыния"},
		{1974, "Гандбол", "ГДР", "Берлин", "Румыния"},
		{1978, "Гандбол", "Дания", "Копенгаген", "ФРГ"},
		{1982, "Гандбол", "ФРГ", "Бонн", "СССР"},
		{1986, "Гандбол", "Швейцария", "Цюрих", "Югославия"},
		{1990, "Гандбол", "Чехословакия", "Прага", "Швеция"},
		{1993, "Гандбол", "Швеция", "Осло", "Россия"},
		{1995, "Гандбол", "Исландия", "Рейкьявик", "Франция"},
		{1997, "Гандбол", "Япония", "Токио", "Россия"},
		{1999, "Гандбол", "Египет", "Каир", "Швеция"},
		{2003, "Гандбол", "Португалия", "Лиссабон", "Хорватия"},
		{2005, "Гандбол", "Тунис", "Тунис", "Испания"},
		{2007, "Гандбол", "Германия", "Берлин", "Германия"},
		{2009, "Гандбол", "Хорватия", "Загреб", "Франция"},
		{2011, "Гандбол", "Швеция", "Осло", "Франция"},
		{2013, "Гандбол", "Испания", "Мадрид", "Испания"},
		{2015, "Гандбол", "Катар", "Доха", "Франция"},
		{2017, "Гандбол", "Франция", "Париж", "Франция"},
		{2019, "Гандбол", "Дания", "Копенгаген", "Дания"},
		{2021, "Гандбол", "Египет", "Каир", "Дания"},
		{2023, "Гандбол", "Польша", "Прага", ""},
		{2025, "Гандбол", "Хорватия", "Загреб", ""},
		{2027, "Гандбол", "Германия", "Берлин", ""},
	}

	t.Run("Put view records to test", func(t *testing.T) {
		batch := make([]istructs.ViewKV, 0)
		for _, c := range championships {
			kv := istructs.ViewKV{}
			kv.Key = app.ViewRecords().KeyBuilder(championshipsView)
			kv.Key.PutInt32("Year", c.year)
			kv.Key.PutString("Sport", c.sport)
			kv.Value = app.ViewRecords().NewValueBuilder(championshipsView)
			kv.Value.PutString("Country", c.country)
			kv.Value.PutString("City", c.city)
			batch = append(batch, kv)

			if c.winner != "" {
				kv := istructs.ViewKV{}
				kv.Key = app.ViewRecords().KeyBuilder(championsView)
				kv.Key.PutInt32("Year", c.year)
				kv.Key.PutString("Sport", c.sport)
				kv.Value = app.ViewRecords().NewValueBuilder(championsView)
				kv.Value.PutString("Winner", c.winner)
				batch = append(batch, kv)
			}
		}

		err := app.ViewRecords().PutBatch(1, batch)
		require.NoError(err)
	})

	t.Run("must ok to read all recs by batch", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 0)
		for _, c := range championships {
			kv := istructs.ViewRecordGetBatchItem{}
			kv.Key = app.ViewRecords().KeyBuilder(championshipsView)
			kv.Key.PutInt32("Year", c.year)
			kv.Key.PutString("Sport", c.sport)
			batch = append(batch, kv)
			if c.winner != "" {
				kv := istructs.ViewRecordGetBatchItem{}
				kv.Key = app.ViewRecords().KeyBuilder(championsView)
				kv.Key.PutInt32("Year", c.year)
				kv.Key.PutString("Sport", c.sport)
				batch = append(batch, kv)
			}
		}

		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.NoError(err)

		i := 0
		for _, c := range championships {
			b := batch[i]
			require.True(b.Ok)
			require.Equal(c.country, b.Value.AsString("Country"))
			require.Equal(c.city, b.Value.AsString("City"))
			i++
			if c.winner != "" {
				b := batch[i]
				require.True(b.Ok)
				require.Equal(c.winner, b.Value.AsString("Winner"))
				i++
			}
		}
	})

	t.Run("must ok to read few records from one view", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 3)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		batch[0].Key.PutString("Sport", "Волейбол")
		batch[1].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[1].Key.PutInt32("Year", 1997)
		batch[1].Key.PutString("Sport", "Гандбол")
		batch[2].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[2].Key.PutInt32("Year", 2075)
		batch[2].Key.PutString("Sport", "Футбол")

		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.NoError(err)

		require.True(batch[0].Ok)
		require.Equal("СССР", batch[0].Value.AsString("Winner"))

		require.True(batch[1].Ok)
		require.Equal("Россия", batch[1].Value.AsString("Winner"))

		require.False(batch[2].Ok)
		require.Equal(appdef.NullQName, batch[2].Value.AsQName(appdef.SystemField_QName))
	})

	t.Run("must fail to read if maximum batch size exceeds", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, maxGetBatchRecordCount+1)
		for i := 0; i < len(batch); i++ {
			batch[i].Key = app.ViewRecords().KeyBuilder(championsView)
			batch[i].Key.PutInt32("Year", int32(i))
			batch[i].Key.PutString("Sport", "Шашки")
		}
		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.ErrorIs(err, ErrMaxGetBatchRecordCountExceeds)
	})

	t.Run("must fail to read if some key build error", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 3)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt64("Year", 1962) // error here
		batch[0].Key.PutString("Sport", "Волейбол")

		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.ErrorIs(err, ErrWrongFieldType)
	})

	t.Run("must fail to read if some key is not valid", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 3)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		// batch[0].Key.PutString("Sport", "Волейбол") // error here

		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.ErrorIs(err, ErrFieldIsEmpty)
	})

	t.Run("must fail to read if storage GetBatch failed", func(t *testing.T) {
		testError := errors.New("test error")

		batch := make([]istructs.ViewRecordGetBatchItem, 1)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		batch[0].Key.PutString("Sport", "Волейбол")

		storage.ScheduleGetError(testError, nil, []byte("Волейбол")) // error here

		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.ErrorIs(err, testError)
	})

	t.Run("must fail to read if storage GetBatch returns damaged data", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 1)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		batch[0].Key.PutString("Sport", "Волейбол")

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, []byte("Волейбол"))

		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.ErrorIs(err, ErrUnknownCodec)
	})

	t.Run("Check IKeyBuilder.Equals", func(t *testing.T) {
		k1 := app.ViewRecords().KeyBuilder(championsView)
		k1.PutInt32("Year", 1962)
		k1.PutString("Sport", "Волейбол")

		require.True(k1.Equals(k1), "KeyBuilder must be equals to itself")

		require.False(k1.Equals(nil), "KeyBuilder must not be equals to nil")

		k2 := app.ViewRecords().KeyBuilder(championsView)
		k2.PutInt32("Year", 1962)
		k2.PutString("Sport", "Волейбол")

		require.True(k1.Equals(k2), "KeyBuilder must be equals if same name and fields")
		require.True(k2.Equals(k1), "KeyBuilder must be equals if same name and fields")

		k2.PutString("Sport", "Гандбол")
		require.False(k1.Equals(k2), "KeyBuilder must not be equals if different clustering fields")
		require.False(k2.Equals(k1), "KeyBuilder must not be equals if different clustering fields")

		k3 := app.ViewRecords().KeyBuilder(championsView)
		k3.PutInt32("Year", 1966)
		k3.PutString("Sport", "Волейбол")

		require.False(k1.Equals(k3), "KeyBuilder must not be equals if different partition fields")
		require.False(k3.Equals(k1), "KeyBuilder must not be equals if different partition fields")

		k4 := app.ViewRecords().KeyBuilder(championshipsView)
		k4.PutInt32("Year", 1962)
		k4.PutString("Sport", "Волейбол")

		require.False(k1.Equals(k4), "KeyBuilder must not be equals if different QNames")
	})
}

func Test_ViewRecordStructure(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	viewName := appdef.NewQName("test", "view")

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")
	t.Run("must be ok to build application", func(t *testing.T) {
		v := adb.AddView(viewName)
		v.Key().PartKey().
			AddField("ValueDateYear", appdef.DataKind_int32)
		v.Key().ClustCols().
			AddField("ValueDateMonth", appdef.DataKind_int32).
			AddField("ValueDateDay", appdef.DataKind_int32).
			AddField("ReportDateYear", appdef.DataKind_int32).
			AddField("ReportDateMonth", appdef.DataKind_int32).
			AddField("ReportDateDay", appdef.DataKind_int32)
		v.Value().
			AddField("ColOffset", appdef.DataKind_int64, true)
	})

	cfg := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app2, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		asp := simpleStorageProvider()
		storage, err := asp.AppStorage(appName)
		require.NoError(err)
		err = cfg.prepare(nil, storage)
		if err != nil {
			panic(err)
		}

		return cfg
	}()

	k1 := newKey(cfg, viewName)
	k1.PutInt32("ValueDateYear", 2023)
	k1.PutInt32("ValueDateMonth", 10)
	k1.PutInt32("ValueDateDay", 27)
	k1.PutInt32("ReportDateYear", 2023)
	k1.PutInt32("ReportDateMonth", 10)
	k1.PutInt32("ReportDateDay", 31)

	err := k1.build()
	require.NoError(err)

	p, c := k1.storeToBytes(0)
	fmt.Printf("%#x\n", p)
	fmt.Printf("%#x\n", c)

	v1 := newValue(cfg, viewName)
	v1.PutInt64("ColOffset", 509)
	require.NoError(v1.build())

	v := v1.storeToBytes()
	fmt.Printf("%#x\n", v)
}
