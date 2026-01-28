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

	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/isequencer"

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

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))

		t.Run("should be ok to build view", func(t *testing.T) {
			view := wsb.AddView(viewName)
			view.Key().PartKey().
				AddField("pk_int8", appdef.DataKind_int8).
				AddField("pk_int16", appdef.DataKind_int16).
				AddField("pk_int32", appdef.DataKind_int32).
				AddField("pk_int64", appdef.DataKind_int64).
				AddField("pk_float32", appdef.DataKind_float32).
				AddField("pk_float64", appdef.DataKind_float64).
				AddField("pk_qname", appdef.DataKind_QName).
				AddField("pk_bool", appdef.DataKind_bool).
				AddRefField("pk_recID").
				AddField("pk_number", appdef.DataKind_float64)
			view.Key().ClustCols().
				AddField("cc_int8", appdef.DataKind_int8).
				AddField("cc_int16", appdef.DataKind_int16).
				AddField("cc_int32", appdef.DataKind_int32).
				AddField("cc_int64", appdef.DataKind_int64).
				AddField("cc_float32", appdef.DataKind_float32).
				AddField("cc_float64", appdef.DataKind_float64).
				AddField("cc_qname", appdef.DataKind_QName).
				AddField("cc_bool", appdef.DataKind_bool).
				AddRefField("cc_recID").
				AddField("cc_number", appdef.DataKind_float64).
				AddField("cc_bytes", appdef.DataKind_bytes, constraints.MaxLen(64))
			view.Value().
				AddField("val_int8", appdef.DataKind_int8, false).
				AddField("val_int16", appdef.DataKind_int16, false).
				AddField("val_string", appdef.DataKind_string, false, constraints.MaxLen(1024))
		})

		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		return cfgs
	}

	appCfgs := appConfigs()
	appCfg := appCfgs.GetConfig(appName)

	appProvider := Provide(appCfgs, iratesce.TestBucketsFactory, testTokensFactory(), teststore.NewStorageProvider(teststore.NewStorage(appName)), isequencer.SequencesTrustLevel_0, nil)
	app, err := appProvider.BuiltIn(appName)
	require.NoError(err)
	require.NotNil(app)

	key := newKey(appCfg, viewName)

	t.Run("should be supported IKeyBuilder", func(t *testing.T) {
		kb := istructs.IKeyBuilder(key)

		require.NotNil(kb)

		kb.PutInt8("pk_int8", 123)
		kb.PutInt16("pk_int16", 12345)
		kb.PutInt32("pk_int32", 1111111)
		kb.PutInt64("pk_int64", 222222222222)
		kb.PutFloat32("pk_float32", 3.333e3)
		kb.PutFloat64("pk_float64", -4.4444e-44)
		kb.PutQName("pk_qname", istructs.QNameForError)
		kb.PutBool("pk_bool", true)
		kb.PutRecordID("pk_recID", istructs.RecordID(5555555))
		kb.PutNumber("pk_number", gojson.Number("1.23456789"))

		kb.PutInt8("cc_int8", 123)
		kb.PutInt16("cc_int16", 12345)
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

		require.EqualValues(123, k.AsInt8("pk_int8"))
		require.EqualValues(12345, k.AsInt16("pk_int16"))
		require.EqualValues(1111111, k.AsInt32("pk_int32"))
		require.EqualValues(222222222222, k.AsInt64("pk_int64"))
		require.EqualValues(3.333e3, k.AsFloat32("pk_float32"))
		require.EqualValues(-4.4444e-44, k.AsFloat64("pk_float64"))
		require.EqualValues(istructs.QNameForError, k.AsQName("pk_qname"))
		require.True(k.AsBool("pk_bool"))
		require.EqualValues(5555555, k.AsRecordID("pk_recID"))
		require.EqualValues(1.23456789, k.AsFloat64("pk_number"))

		require.EqualValues(123, k.AsInt8("cc_int8"))
		require.EqualValues(12345, k.AsInt16("cc_int16"))
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
			view := appdef.View(appCfg.AppDef.Type, viewName)
			cnt := 0
			k.Fields(func(iField appdef.IField) bool {
				require.NotNil(view.Key().Field(iField.Name()), "unknown field name passed in callback from IKey.FieldNames(): %q", iField.Name())
				cnt++
				return true
			})
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

	t.Run("should be supported IKey interface", func(t *testing.T) { testIKey(t, key) })

	t.Run("should be ok to load/store key to bytes", func(t *testing.T) {
		p, c := key.storeToBytes(0)
		require.NotEmpty(p)
		require.NotEmpty(c)

		dupe := newKey(appCfg, viewName)
		dupe.partRow.copyFrom(&key.partRow)
		require.NoError(dupe.loadFromBytes(c))

		t.Run("should be ok to call IKey members", func(t *testing.T) { testIKey(t, dupe) })

		require.True(key.Equals(dupe))
	})

	t.Run("should be ok IValueBuilder.ToBytes()", func(t *testing.T) {
		vb := newValue(appCfg, viewName)
		vb.PutInt8("val_int8", 123)
		vb.PutInt16("val_int16", 12345)
		vb.PutString("val_string", "test string")

		b, err := vb.ToBytes()
		require.NoError(err)
		require.NotEmpty(b)

		dupe := newValue(appCfg, viewName)
		err = dupe.loadFromBytes(b)
		require.NoError(err)
		require.EqualValues(123, dupe.AsInt8("val_int8"))
		require.EqualValues(12345, dupe.AsInt16("val_int16"))
		require.EqualValues(12345, dupe.AsInt16("val_int16"))
		require.EqualValues("test string", dupe.AsString("val_string"))
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

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		t.Run("should be ok to build application", func(t *testing.T) {
			view := wsb.AddView(appdef.NewQName("test", "viewDrinks"))
			view.Key().PartKey().
				AddField("partitionKey1", appdef.DataKind_int64)
			view.Key().ClustCols().
				AddField("clusteringColumn1", appdef.DataKind_int64).
				AddField("clusteringColumn2", appdef.DataKind_bool).
				AddField("clusteringColumn3", appdef.DataKind_string, constraints.MaxLen(64))
			view.Value().
				AddField("id", appdef.DataKind_int64, true).
				AddField("name", appdef.DataKind_string, true).
				AddField("active", appdef.DataKind_bool, true)

			otherView := wsb.AddView(appdef.NewQName("test", "otherView"))
			otherView.Key().PartKey().
				AddField("partitionKey1", appdef.DataKind_QName)
			otherView.Key().ClustCols().
				AddField("clusteringColumn1", appdef.DataKind_float32).
				AddField("clusteringColumn2", appdef.DataKind_float64).
				AddField("clusteringColumn3", appdef.DataKind_bytes, constraints.MaxLen(128))
			otherView.Value().
				AddField("valueField1", appdef.DataKind_int64, false)
		})

		cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		return cfgs
	}

	appCfgs := appConfigs()
	appCfg := appCfgs.GetConfig(istructs.AppQName_test1_app1)
	p := Provide(appCfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
	app, err := p.BuiltIn(istructs.AppQName_test1_app1)
	require.NoError(err)
	viewRecords := app.ViewRecords()

	t.Run("should be ok put records one-by-one", func(t *testing.T) {
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

	t.Run("should be ok to read one (!) record by WSID = 1", func(t *testing.T) {
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

	t.Run("should be ok batch put", func(t *testing.T) {
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

	t.Run("should be ok to read three record from WSID = 3 with correct order", func(t *testing.T) {
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

	t.Run("should be ok to read two records by short clustering key and one by full", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)

		t.Run("should be ok to read one records by short clustering key", func(t *testing.T) {
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

		t.Run("should be ok to read one records by short «c» clustering key", func(t *testing.T) {
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

		t.Run("should be ok to read one record by long «cid» clustering key", func(t *testing.T) {
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

		t.Run("should be no records by not existing clustering key", func(t *testing.T) {
			kb.PutString("clusteringColumn3", "tofu")
			counter := 0
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				return nil
			})
			require.NoError(err)
			require.Equal(0, counter)
		})

		t.Run("should be ok to read one records by short «s» clustering key. Old style key filling", func(t *testing.T) {
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

	t.Run("should be ok to get exists record", func(t *testing.T) {
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

	t.Run("should be error to get non exists record", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "tofu")

		value, err := viewRecords.Get(2, kb)
		require.ErrorIs(err, istructs.ErrRecordNotFound)
		require.NotNil(value)
	})

	t.Run("should be ok to call UpdateValueBuilder", func(t *testing.T) {
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

	t.Run("should be errors in build key", func(t *testing.T) {

		require.Panics(func() { _ = viewRecords.KeyBuilder(appdef.NullQName) },
			require.Is(ErrNameMissedError, "Should panics if key type missed"))

		require.Panics(func() { _ = viewRecords.KeyBuilder(istructs.QNameForError) },
			require.Is(ErrNameNotFoundError), require.Has(istructs.QNameForError))
		require.Panics(func() { _ = viewRecords.KeyBuilder(appdef.NewQName("test", "unknownDrinks")) },
			require.Is(ErrNameNotFoundError), require.Has("test.unknownDrinks"))

		require.Panics(func() { _ = viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks_Value")) },
			require.Is(ErrNameNotFoundError), require.Has("test.viewDrinks_Value"))

		t.Run("if wrong partition key type", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			pk := kb.PartitionKey()
			pk.PutQName(appdef.SystemField_QName, appdef.NewQName("test", "viewDrinks_Value"))
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.Error(err, require.Is(ErrUnableToUpdateSystemFieldError), require.HasAll("test.viewDrinks", appdef.SystemField_QName))
		})

		t.Run("if wrong clustering columns type", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			pk := kb.PartitionKey()
			pk.PutInt64("partitionKey1", 1)
			cc := kb.ClusteringColumns()
			cc.PutQName(appdef.SystemField_QName, appdef.NewQName("test", "viewDrinks_Value"))
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.Error(err, require.Is(ErrUnableToUpdateSystemFieldError), require.HasAll("test.viewDrinks", appdef.SystemField_QName))
		})

		t.Run("if holes in clustering column", func(t *testing.T) {
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
			require.Error(err, require.Is(ErrFieldIsEmptyError),
				require.HasAll("test.viewDrinks", "clusteringColumn2"))
			require.Zero(cnt)
		})

		t.Run("if wrong value type", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.PutInt64("clusteringColumn1", 100)
			kb.PutBool("clusteringColumn2", true)
			kb.PutString("clusteringColumn3", "soda")

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
			vb.PutQName(appdef.SystemField_QName, appdef.NewQName("test", "viewDrinks_ClusteringColumns"))

			err := viewRecords.Put(1, kb, vb)
			require.Error(err, require.Is(ErrUnableToUpdateSystemFieldError), require.HasAll("test.viewDrinks", appdef.SystemField_QName))
		})

		t.Run("if empty partition key", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))

			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has("test.viewDrinks"))

			validateErr := validateErrorf(0, "")
			require.ErrorAs(err, &validateErr)
			require.Equal(ECode_EmptyData, validateErr.Code())

			_, err = viewRecords.Get(1, kb)
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has("test.viewDrinks"))
		})

		t.Run("if put with empty clustering columns key", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))
			vb.PutInt64("id", 1)
			vb.PutString("name", "tea")
			vb.PutBool("active", true)

			err := viewRecords.Put(1, kb, vb)
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has("test.viewDrinks"))

			validateErr := validateErrorf(0, "")
			require.ErrorAs(err, &validateErr)
			require.Equal(ECode_EmptyData, validateErr.Code())
		})

		t.Run("if put with wrong fields in key", func(t *testing.T) {

			t.Run("should be put error", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
				kb.PutBool("errorField", true)
				err := viewRecords.Put(1, kb, nil)
				require.Error(err, require.Is(ErrNameNotFoundError), require.Has("errorField"))

				t.Run("should be error IKeyBuilder.ToBytes()", func(t *testing.T) {
					pk, cc, err := kb.ToBytes(0)
					require.Error(err, require.Is(ErrNameNotFoundError), require.Has("errorField"))
					require.Empty(pk)
					require.Empty(cc)
				})
			})

			t.Run("if read with invalid key", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
				kb.PutBool("errorField", true)
				err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
					return nil
				})
				require.Error(err, "if Read() with invalid key",
					require.Is(ErrNameNotFoundError), require.Has("errorField"))

				_, err = viewRecords.Get(1, kb)
				require.Error(err, "if Get() with invalid key",
					require.Is(ErrNameNotFoundError), require.Has("errorField"))
			})

		})

		t.Run("if get with wrong fields in partition key", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PartitionKey().PutBool("errorField", true)
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.Error(err, "if Read() with invalid PK",
				require.Is(ErrNameNotFoundError), require.Has("errorField"))

			_, err = viewRecords.Get(1, kb)
			require.Error(err, "if Get() with invalid PK",
				require.Is(ErrNameNotFoundError), require.Has("errorField"))
		})

		t.Run("if get with wrong fields in clustering columns", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.ClusteringColumns().PutBytes("errorField", []byte{1, 2, 3})
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.Error(err, "if Read() with invalid CCols",
				require.Is(ErrNameNotFoundError), require.Has("errorField"))

			_, err = viewRecords.Get(1, kb)
			require.Error(err, "if Get() with invalid CCols",
				require.Is(ErrNameNotFoundError), require.Has("errorField"))
		})
	})

	t.Run("should be errors in build value", func(t *testing.T) {

		t.Run("should be panics", func(t *testing.T) {
			require.Panics(func() { _ = viewRecords.NewValueBuilder(appdef.NullQName) },
				"if view name missed",
				require.Is(ErrNameMissedError))

			require.Panics(func() { _ = viewRecords.NewValueBuilder(appdef.NewQName("test", "unknownDrinks")) },
				"if view not found",
				require.Is(ErrNameNotFoundError), require.Has("test.unknownDrinks"))

			require.Panics(func() { _ = viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks_PartitionKey")) },
				"if wrong (not a view) type passed",
				require.Is(ErrNameNotFoundError), require.Has("test.viewDrinks_PartitionKey"))

			t.Run("if wrong existing value type specified", func(t *testing.T) {
				exists := newValue(appCfg, appdef.NewQName("test", "otherView"))
				require.Panics(func() {
					_ = viewRecords.UpdateValueBuilder(appdef.NewQName("test", "viewDrinks"), exists)
				}, require.Is(ErrWrongTypeError), require.Has("test.otherView"))
			})
		})

		t.Run("if put with empty value", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.PutInt64("clusteringColumn1", 100)
			kb.PutBool("clusteringColumn2", true)
			kb.PutString("clusteringColumn3", "null")

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "viewDrinks"))

			err := viewRecords.Put(1, kb, vb)
			require.Error(err, require.Is(ErrFieldIsEmptyError),
				require.HasAll("id", "name", "active"))

			validateErr := validateErrorf(0, "")
			require.ErrorAs(err, &validateErr)
			require.Equal(ECode_EmptyData, validateErr.Code())
		})

		t.Run("if put with invalid value", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(appdef.NewQName("test", "otherView"))
			kb.PutQName("partitionKey1", appdef.NullQName)
			kb.PutFloat32("clusteringColumn1", 44.4)
			kb.PutFloat64("clusteringColumn2", 64.4)
			kb.PutBytes("clusteringColumn3", []byte("TEST"))

			vb := viewRecords.NewValueBuilder(appdef.NewQName("test", "otherView"))
			vb.PutQName("unknownField", appdef.NullQName)

			err := viewRecords.Put(1, kb, vb)
			require.Error(err, "if put with unknown field",
				require.Is(ErrNameNotFoundError), require.Has("unknownField"))

			t.Run("should be error IValueBuilder.ToBytes()", func(t *testing.T) {
				v, err := vb.ToBytes()
				require.Error(err, "if ToBytes() with unknown field",
					require.Is(ErrNameNotFoundError), require.Has("unknownField"))
				require.Empty(v)
			})
		})

		t.Run("if key and value are from different views", func(t *testing.T) {
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
			require.Error(err, require.Is(ErrWrongTypeError), require.Has("test.viewDrinks"))
		})

		t.Run("if put batch must with invalid key-value item", func(t *testing.T) {
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

			const failedWSID = istructs.WSID(7)
			err := viewRecords.PutBatch(failedWSID, batch)
			require.Error(err, require.Is(ErrNameNotFoundError),
				require.Has("test.viewDrinks"), require.Has("errorField"))

			t.Run("should be no reads if put batch failed", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
				kb.PutInt64("partitionKey1", 7)

				require.NoError(viewRecords.Read(context.Background(), failedWSID, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
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

		const badCodec byte = 255

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = badCodec /* error here */ }, nil, c)
		_, err := viewRecords.Get(2, kb)
		require.Error(err, require.Is(ErrUnknownCodecError), require.Has(badCodec))

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = badCodec /* error here */ }, nil, c)
		err = viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) { return nil })
		require.Error(err, require.Is(ErrUnknownCodecError), require.Has(badCodec))
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

		require.Panics(func() { _ = vb.Build() }, require.Is(ErrWrongFieldTypeError), require.Has("id"))
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

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		t.Run("should be ok to build application", func(t *testing.T) {
			view := wsb.AddView(appdef.MustParseQName(viewName))
			view.Key().PartKey().
				AddField("pk1", appdef.DataKind_int64)
			view.Key().ClustCols().
				AddField("cc1", appdef.DataKind_int64).
				AddField("cc2", appdef.DataKind_string, constraints.MaxLen(64))
			view.Value().
				AddField("v1", appdef.DataKind_float32, true).
				AddField("v2", appdef.DataKind_string, true)
		})
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		return cfgs
	}()

	app, err := Provide(appCfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil).BuiltIn(appName)
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

	t.Run("should be error", func(t *testing.T) {
		var err error
		t.Run("if wrong view name", func(t *testing.T) {
			json := make(map[appdef.FieldName]any)

			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has(appdef.SystemField_QName))

			json[appdef.SystemField_QName] = appdef.NullQName.String()
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has(appdef.SystemField_QName))

			json[appdef.SystemField_QName] = 123
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrWrongFieldTypeError), require.Has(appdef.SystemField_QName))

			json[appdef.SystemField_QName] = `naked 🔫`
			err = app.ViewRecords().PutJSON(1, json)
			require.ErrorIs(err, appdef.ErrConvertError)
			require.ErrorContains(err, appdef.SystemField_QName)

			json[appdef.SystemField_QName] = `test.unknown`
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrNameNotFoundError), require.Has(`test.unknown`))
		})

		t.Run("if key errors", func(t *testing.T) {
			json := make(map[appdef.FieldName]any)
			json[appdef.SystemField_QName] = viewName

			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has("pk1"))

			json["pk1"] = "error value"
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrWrongFieldTypeError), require.Has("pk1"))

			json["pk1"] = gojson.Number("1")
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has("cc1"))

			json["pk1"] = gojson.Number("1")
			json["cc1"] = gojson.Number("2")
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrFieldIsEmptyError), require.Has("cc2"))
		})

		t.Run("if value errors", func(t *testing.T) {
			json := make(map[appdef.FieldName]any)
			json[appdef.SystemField_QName] = viewName
			json["pk1"] = gojson.Number("1")
			json["cc1"] = gojson.Number("2")
			json["cc2"] = `test sort`

			json["unknownField"] = `value`
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrNameNotFoundError), require.Has("unknownField"))

			delete(json, "unknownField")
			json["v1"] = `value`
			err = app.ViewRecords().PutJSON(1, json)
			require.Error(err, require.Is(ErrWrongFieldTypeError), require.Has(err, "v1"))
		})
	})
}

func Test_LoadStoreViewRecord_Bytes(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	viewName := appdef.NewQName("test", "view")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
	wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
	wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
	t.Run("should be ok to build application", func(t *testing.T) {
		v := wsb.AddView(viewName)
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
			AddField("cc_bytes", appdef.DataKind_bytes, constraints.MaxLen(8))
		v.Value().
			AddField("vf_int32", appdef.DataKind_int32, true).
			AddField("vf_int64", appdef.DataKind_int64, false).
			AddField("vf_float32", appdef.DataKind_float32, false).
			AddField("vf_float64", appdef.DataKind_float64, false).
			AddField("vf_bytes", appdef.DataKind_bytes, false, constraints.MaxLen(1024)).
			AddField("vf_string", appdef.DataKind_string, false, constraints.Pattern(`^\w+$`)).
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
		for i := range len(c) - 4 { // 4 - is length of variable bytes "test" that can be truncated with impunity
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
		for i := range v {
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

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		t.Run("should be ok to build application", func(t *testing.T) {
			v := wsb.AddView(appdef.NewQName("test", "viewDrinks"))
			v.Key().PartKey().
				AddField("partitionKey1", appdef.DataKind_int64)
			v.Key().ClustCols().
				AddField("clusteringColumn1", appdef.DataKind_QName).
				AddRefField("clusteringColumn2")
			v.Value().
				AddField("id", appdef.DataKind_int64, true).
				AddField("name", appdef.DataKind_string, true).
				AddField("active", appdef.DataKind_bool, true)

			_ = wsb.AddObject(appdef.NewQName("test", "obj1"))
		})

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		return cfgs
	}

	p := Provide(appConfigs(), iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)
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
	t.Run("should be ok to read single item", func(t *testing.T) {
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

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
	wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
	wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
	t.Run("should be ok to build application", func(t *testing.T) {
		v := wsb.AddView(championshipsView)
		v.Key().PartKey().
			AddField("Year", appdef.DataKind_int32)
		v.Key().ClustCols().
			AddField("Sport", appdef.DataKind_string, constraints.MaxLen(64))
		v.Value().
			AddField("Country", appdef.DataKind_string, true).
			AddField("City", appdef.DataKind_string, false)

		v = wsb.AddView(championsView)
		v.Key().PartKey().
			AddField("Year", appdef.DataKind_int32)
		v.Key().ClustCols().
			AddField("Sport", appdef.DataKind_string, constraints.MaxLen(64))
		v.Value().
			AddField("Winner", appdef.DataKind_string, true)
	})

	storage := teststore.NewStorage(appName)
	storageProvider := teststore.NewStorageProvider(storage)

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	type championship struct {
		year                 int32
		sport, country, city string
		winner               string
	}
	var championships = []championship{
		{1949, "Volleyball", "Czechoslovakia", "Prague", "USSR"},
		{1952, "Volleyball", "USSR", "Moscow", "USSR"},
		{1956, "Volleyball", "France", "Paris", "Czechoslovakia"},
		{1960, "Volleyball", "Brazil", "Rio de Janeiro", "USSR"},
		{1962, "Volleyball", "USSR", "Moscow", "USSR"},
		{1966, "Volleyball", "Czechoslovakia", "Prague", "Czechoslovakia"},
		{1970, "Volleyball", "Bulgaria", "Sofia", "GDR"},
		{1974, "Volleyball", "Mexico", "Mexico", "Poland"},
		{1978, "Volleyball", "Italy", "Rome", "USSR"},
		{1982, "Volleyball", "Argentina", "Buenos Aires", "USSR"},
		{1986, "Volleyball", "France", "Paris", "USA"},
		{1990, "Volleyball", "Brazil", "Rio de Janeiro", "Italy"},
		{1994, "Volleyball", "Greece", "Athens", "Italy"},
		{1998, "Volleyball", "Japan", "Tokyo", "Italy"},
		{2002, "Volleyball", "Argentina", "Buenos Aires", "Brazil"},
		{2006, "Volleyball", "Japan", "Tokyo", "Brazil"},
		{2010, "Volleyball", "Italy", "Rome", "Brazil"},
		{2014, "Volleyball", "Poland", "Katowice", "Poland"},
		{2018, "Volleyball", "Italy", "Rome", "Poland"},
		{2022, "Volleyball", "Russia", "Moscow", ""},

		{1938, "Handball", "Germany", "Berlin", "Germany"},
		{1942, "Handball", "Sweden", "Oslo", "Sweden"},
		{1958, "Handball", "GDR", "Berlin", "Sweden"},
		{1961, "Handball", "FRG", "Bonn", "Romania"},
		{1964, "Handball", "Czechoslovakia", "Prague", "Romania"},
		{1967, "Handball", "Sweden", "Oslo", "Czechoslovakia"},
		{1970, "Handball", "France", "Paris", "Romania"},
		{1974, "Handball", "GDR", "Berlin", "Romania"},
		{1978, "Handball", "Denmark", "Copenhagen", "FRG"},
		{1982, "Handball", "FRG", "Bonn", "USSR"},
		{1986, "Handball", "Switzerland", "Zurich", "Yugoslavia"},
		{1990, "Handball", "Czechoslovakia", "Prague", "Sweden"},
		{1993, "Handball", "Sweden", "Oslo", "Russia"},
		{1995, "Handball", "Iceland", "Reykjavik", "France"},
		{1997, "Handball", "Japan", "Tokyo", "Russia"},
		{1999, "Handball", "Egypt", "Cairo", "Sweden"},
		{2003, "Handball", "Portugal", "Lisbon", "Croatia"},
		{2005, "Handball", "Tunisia", "Tunisia", "Spain"},
		{2007, "Handball", "Germany", "Berlin", "Germany"},
		{2009, "Handball", "Croatia", "Zagreb", "France"},
		{2011, "Handball", "Sweden", "Oslo", "France"},
		{2013, "Handball", "Spain", "Madrid", "Spain"},
		{2015, "Handball", "Qatar", "Doha", "France"},
		{2017, "Handball", "France", "Paris", "France"},
		{2019, "Handball", "Denmark", "Copenhagen", "Denmark"},
		{2021, "Handball", "Egypt", "Cairo", "Denmark"},
		{2023, "Handball", "Poland", "Prague", ""},
		{2025, "Handball", "Croatia", "Zagreb", ""},
		{2027, "Handball", "Germany", "Berlin", ""},
	}

	t.Run("should be ok to put view records", func(t *testing.T) {
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

	t.Run("should be ok to read all recs by batch", func(t *testing.T) {
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

	t.Run("should be ok to read few records from one view", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 3)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		batch[0].Key.PutString("Sport", "Volleyball")
		batch[1].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[1].Key.PutInt32("Year", 1997)
		batch[1].Key.PutString("Sport", "Handball")
		batch[2].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[2].Key.PutInt32("Year", 2075)
		batch[2].Key.PutString("Sport", "Football")

		err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
		require.NoError(err)

		require.True(batch[0].Ok)
		require.Equal("USSR", batch[0].Value.AsString("Winner"))

		require.True(batch[1].Ok)
		require.Equal("Russia", batch[1].Value.AsString("Winner"))

		require.False(batch[2].Ok)
		require.Equal(appdef.NullQName, batch[2].Value.AsQName(appdef.SystemField_QName))
	})

	t.Run("should be error", func(t *testing.T) {

		t.Run("if maximum batch size exceeds", func(t *testing.T) {
			const tooGig = maxGetBatchRecordCount + 1
			batch := make([]istructs.ViewRecordGetBatchItem, tooGig)
			for i := range batch {
				batch[i].Key = app.ViewRecords().KeyBuilder(championsView)
				batch[i].Key.PutInt32("Year", int32(i))
				batch[i].Key.PutString("Sport", "Шашки")
			}
			err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
			require.Error(err, require.Is(ErrMaxGetBatchSizeExceedsError), require.Has(tooGig))
		})

		t.Run("if key build error", func(t *testing.T) {
			batch := make([]istructs.ViewRecordGetBatchItem, 3)
			batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
			batch[0].Key.PutInt64("Year", 1962) // error here
			batch[0].Key.PutString("Sport", "Volleyball")

			err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
			require.Error(err, require.Is(ErrWrongFieldTypeError), require.Has("Year"))
		})

		t.Run("if key is not valid", func(t *testing.T) {
			batch := make([]istructs.ViewRecordGetBatchItem, 3)
			batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
			batch[0].Key.PutInt32("Year", 1962)
			// batch[0].Key.PutString("Sport", "Volleyball") // error here

			err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
			require.Error(err, require.Is(ErrFieldIsEmptyError),
				require.HasAll(championsView, "Sport"))
		})

		t.Run("if storage GetBatch failed", func(t *testing.T) {
			testError := errors.New("test error")

			batch := make([]istructs.ViewRecordGetBatchItem, 1)
			batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
			batch[0].Key.PutInt32("Year", 1962)
			batch[0].Key.PutString("Sport", "Volleyball")

			storage.ScheduleGetError(testError, nil, []byte("Volleyball")) // error here

			err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
			require.ErrorIs(err, testError)
		})

		t.Run("if GetBatch returns damaged data", func(t *testing.T) {
			batch := make([]istructs.ViewRecordGetBatchItem, 1)
			batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
			batch[0].Key.PutInt32("Year", 1962)
			batch[0].Key.PutString("Sport", "Volleyball")

			storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, []byte("Volleyball"))

			err := app.ViewRecords().(*appViewRecords).GetBatch(1, batch)
			require.ErrorIs(err, ErrUnknownCodecError)
		})
	})

	t.Run("IKeyBuilder.Equals should be", func(t *testing.T) {
		k1 := app.ViewRecords().KeyBuilder(championsView)
		k1.PutInt32("Year", 1962)
		k1.PutString("Sport", "Volleyball")

		require.True(k1.Equals(k1), "equals to itself")

		require.False(k1.Equals(nil), "not equals to nil")

		k2 := app.ViewRecords().KeyBuilder(championsView)
		k2.PutInt32("Year", 1962)
		k2.PutString("Sport", "Volleyball")

		require.True(k1.Equals(k2), "equals if same name and fields")
		require.True(k2.Equals(k1), "equals if same name and fields")

		k2.PutString("Sport", "Handball")
		require.False(k1.Equals(k2), "not equals if different clustering fields")
		require.False(k2.Equals(k1), "not equals if different clustering fields")

		k3 := app.ViewRecords().KeyBuilder(championsView)
		k3.PutInt32("Year", 1966)
		k3.PutString("Sport", "Volleyball")

		require.False(k1.Equals(k3), "not equals if different partition fields")
		require.False(k3.Equals(k1), "not equals if different partition fields")

		k4 := app.ViewRecords().KeyBuilder(championshipsView)
		k4.PutInt32("Year", 1962)
		k4.PutString("Sport", "Volleyball")

		require.False(k1.Equals(k4), "not be equals if different QNames")
	})
}

func Test_ViewRecordStructure(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	viewName := appdef.NewQName("test", "view")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	t.Run("should be ok to build application", func(t *testing.T) {
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		v := wsb.AddView(viewName)
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
