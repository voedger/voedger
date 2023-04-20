/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/schemas"
)

// TestCore_ViewRecords: test https://dev.heeus.io/launchpad/#!14470
func TestCore_ViewRecords(t *testing.T) {
	require := require.New(t)

	storage := teststore.NewTestStorage()
	storageProvider := teststore.NewTestStorageProvider(storage)

	appConfigs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)

		schemas := schemas.NewSchemaCache()
		t.Run("must be ok to build schemas", func(t *testing.T) {
			viewSchema := schemas.AddView(istructs.NewQName("test", "viewDrinks"))
			viewSchema.
				AddPartField("partitionKey1", istructs.DataKind_int64).
				AddClustColumn("clusteringColumn1", istructs.DataKind_int64).
				AddClustColumn("clusteringColumn2", istructs.DataKind_bool).
				AddClustColumn("clusteringColumn3", istructs.DataKind_string).
				AddValueField("id", istructs.DataKind_int64, true).
				AddValueField("name", istructs.DataKind_string, true).
				AddValueField("active", istructs.DataKind_bool, true)

			otherViewSchema := schemas.AddView(istructs.NewQName("test", "otherView"))
			otherViewSchema.
				AddPartField("partitionKey1", istructs.DataKind_QName).
				AddClustColumn("clusteringColumn1", istructs.DataKind_float32).
				AddClustColumn("clusteringColumn2", istructs.DataKind_float64).
				AddClustColumn("clusteringColumn3", istructs.DataKind_bytes).
				AddValueField("valueField1", istructs.DataKind_int64, false)
		})

		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		return cfgs
	}

	appCfgs := appConfigs()
	appCfg := appCfgs.GetConfig(istructs.AppQName_test1_app1)
	p := Provide(appCfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
	app, err := p.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	viewRecords := app.ViewRecords()

	t.Run("must be ok put records one-by-one", func(t *testing.T) {
		entries := []entryType{
			newEntry(viewRecords, 1, 100, true, "soda", 1, "Cola"),
			newEntry(viewRecords, 1, 100, true, "soda", 2, "Cola light"), // dupe, must override previous name
			newEntry(viewRecords, 2, 100, true, "soda", 2, "Pepsi"),
			newEntry(viewRecords, 2, 100, true, "sidr", 2, "Apple sidr"),
		}
		for _, e := range entries {
			err := viewRecords.Put(e.wsid, e.key, e.value)
			require.NoError(err)
		}
	})

	t.Run("Should read one (!) record by WSID = 1", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
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
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
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
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)

		t.Run("Should read two records by short clustering key", func(t *testing.T) {
			counter := 0
			val_name := ""
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_name += value.AsString("name")
				return nil
			})
			require.NoError(err)
			require.Equal(2, counter)
			require.Equal("Apple sidrPepsi", val_name)
		})

		t.Run("Should read two records by short masked clustering key", func(t *testing.T) {
			kb.PutString("clusteringColumn3", "s")
			counter := 0
			val_name := ""
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_name += value.AsString("name")
				return nil
			})
			require.NoError(err)
			require.Equal(2, counter)
			require.Equal("Apple sidrPepsi", val_name)
		})

		t.Run("Should read one record by long masked clustering key", func(t *testing.T) {
			kb.PutString("clusteringColumn3", "si")
			counter := 0
			val_name := ""
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_name += value.AsString("name")
				return nil
			})
			require.NoError(err)
			require.Equal(1, counter)
			require.Equal("Apple sidr", val_name)
		})

		t.Run("Should no read records by not existing clustering key", func(t *testing.T) {
			kb.PutString("clusteringColumn3", "simba")
			counter := 0
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				return nil
			})
			require.NoError(err)
			require.Equal(0, counter)
		})

		t.Run("Should read two records by short masked clustering key. ***Old style key filling", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
			kb.PartitionKey().PutInt64("partitionKey1", 2)
			kb.ClusteringColumns().PutInt64("clusteringColumn1", 100)
			kb.ClusteringColumns().PutBool("clusteringColumn2", true)
			kb.ClusteringColumns().PutString("clusteringColumn3", "s")
			counter := 0
			val_name := ""
			err := viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				counter++
				val_name += value.AsString("name")
				return nil
			})
			require.NoError(err)
			require.Equal(2, counter)
			require.Equal("Apple sidrPepsi", val_name)
		})
	})

	t.Run("get exists record must be ok", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "sidr")

		value, err := viewRecords.Get(2, kb)
		require.NoError(err)
		require.Equal(int64(2), value.AsInt64("id"))
		require.Equal("Apple sidr", value.AsString("name"))
		require.True(value.AsBool("active"))
	})

	t.Run("get not exists record must be available", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "sake")

		value, err := viewRecords.Get(2, kb)
		require.ErrorIs(err, ErrRecordNotFound)
		require.NotNil(value)
	})

	t.Run("Test UpdateValueBuilder", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
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

		vb := viewRecords.UpdateValueBuilder(istructs.NewQName("test", "viewDrinks"), oldValue)
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

		t.Run("Must have panic if key schema missed", func(t *testing.T) {
			require.Panics(func() { _ = viewRecords.KeyBuilder(istructs.NullQName) })
		})

		t.Run("Must have panic if invalid key schema name", func(t *testing.T) {
			require.Panics(func() { _ = viewRecords.KeyBuilder(istructs.QNameForError) })
			require.Panics(func() { _ = viewRecords.KeyBuilder(istructs.NewQName("test", "mismDrinks")) })
		})

		t.Run("Must have panic if invalid key schema kind", func(t *testing.T) {
			require.Panics(func() { _ = viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks_Value")) })
		})

		t.Run("Must have error if wrong partition key schema", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
			pk := kb.PartitionKey()
			pk.PutQName(istructs.SystemField_QName, istructs.NewQName("test", "viewDrinks_Value"))
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrSchemaChanged)
		})

		t.Run("Must have error if wrong clustering columns schema", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
			pk := kb.PartitionKey()
			pk.PutInt64("partitionKey1", 1)
			cc := kb.ClusteringColumns()
			cc.PutQName(istructs.SystemField_QName, istructs.NewQName("test", "viewDrinks_Value"))
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrSchemaChanged)
		})

		t.Run("Must have error if wrong value schema", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.PutInt64("clusteringColumn1", 100)
			kb.PutBool("clusteringColumn2", true)
			kb.PutString("clusteringColumn3", "soda")

			vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))
			vb.PutQName(istructs.SystemField_QName, istructs.NewQName("test", "viewDrinks_ClusteringColumns"))

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrSchemaChanged)
		})

		t.Run("Must have error if empty partition key", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
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
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)

			vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))
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
				kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
				kb.PutBool("errorField", true)
				err := viewRecords.Put(1, kb, nil)
				require.ErrorIs(err, ErrNameNotFound)
			})

			t.Run("Must read and error", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
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
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
			kb.PartitionKey().PutBool("errorField", true)
			err := viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
				return nil
			})
			require.ErrorIs(err, ErrNameNotFound)

			_, err = viewRecords.Get(1, kb)
			require.ErrorIs(err, ErrNameNotFound)
		})

		t.Run("Must have error if wrong fields in clustering columns", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
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

		t.Run("Must have panic if value schema missed", func(t *testing.T) {
			require.Panics(func() { _ = viewRecords.NewValueBuilder(istructs.NullQName) })
		})

		t.Run("Must have panic if unknown value schema specified", func(t *testing.T) {
			require.Panics(func() { _ = viewRecords.NewValueBuilder(istructs.NewQName("test", "mismDrinks")) })
		})

		t.Run("Must have panic if wrong value schema specified", func(t *testing.T) {
			require.Panics(func() { _ = viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks_PartitionKey")) })
		})

		t.Run("Must have panic if wrong existing value schema specified", func(t *testing.T) {
			exists := newValue(appCfg, istructs.NewQName("test", "otherView"))
			require.Panics(func() {
				_ = viewRecords.UpdateValueBuilder(istructs.NewQName("test", "viewDrinks"), exists)
			})
		})

		t.Run("Must have error for put if empty value", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
			kb.PutInt64("partitionKey1", 1)
			kb.PutInt64("clusteringColumn1", 100)
			kb.PutBool("clusteringColumn2", true)
			kb.PutString("clusteringColumn3", "null")

			vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrNameNotFound)

			validateErr := validateErrorf(0, "")
			require.ErrorAs(err, &validateErr)
			require.Equal(ECode_EmptyData, validateErr.Code())
		})

		t.Run("Must have error if errors in value", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "otherView"))
			kb.PutQName("partitionKey1", istructs.NullQName)
			kb.PutFloat32("clusteringColumn1", 44.4)
			kb.PutFloat64("clusteringColumn2", 64.4)
			kb.PutBytes("clusteringColumn3", []byte("TEST"))

			vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "otherView"))
			vb.PutQName("unknownField", istructs.NullQName)

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrNameNotFound)
		})

		t.Run("Must have error if key and value are from different views", func(t *testing.T) {
			kb := viewRecords.KeyBuilder(istructs.NewQName("test", "otherView"))
			kb.PutQName("partitionKey1", istructs.NullQName)
			kb.PutFloat32("clusteringColumn1", 44.4)
			kb.PutFloat64("clusteringColumn2", 64.4)
			kb.PutBytes("clusteringColumn3", []byte("TEST"))

			vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))
			vb.PutInt64("id", 1)
			vb.PutString("name", "baykal")
			vb.PutBool("active", true)

			err := viewRecords.Put(1, kb, vb)
			require.ErrorIs(err, ErrWrongSchema)
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

			t.Run("put batch failed; no record from WSID = 7 must be readed", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
				kb.PutInt64("partitionKey1", 7)

				require.NoError(viewRecords.Read(context.Background(), 7, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
					require.Fail("if put batch failed then no records must be readed")
					return nil
				}))
			})
		})

		t.Run("Must have not error if all is ok", func(t *testing.T) {

			t.Run("vulgaris case", func(t *testing.T) {
				kb := viewRecords.KeyBuilder(istructs.NewQName("test", "otherView"))
				kb.PutQName("partitionKey1", istructs.QNameForError)
				kb.PutFloat32("clusteringColumn1", 44.4)
				kb.PutFloat64("clusteringColumn2", 64.4)
				kb.PutBytes("clusteringColumn3", []byte("TEST"))

				vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "otherView"))
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
				kb := viewRecords.KeyBuilder(istructs.NewQName("test", "otherView"))
				kb.PutQName("partitionKey1", istructs.QNameForError)
				kb.PutFloat32("clusteringColumn1", 44.4)
				kb.PutFloat64("clusteringColumn2", 64.4)
				kb.PutBytes("clusteringColumn3", []byte{0xFF, 0x1, 0x2})

				vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "otherView"))
				vb.PutInt64("valueField1", 7)

				err := viewRecords.Put(1, kb, vb)
				require.NoError(err)

				readKey := viewRecords.KeyBuilder(istructs.NewQName("test", "otherView"))
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
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 100)
		kb.PutBool("clusteringColumn2", true)
		kb.PutString("clusteringColumn3", "sidr")

		c := utils.PrefixBytes([]byte("sidr"), int64(100), true)

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, c)
		_, err := viewRecords.Get(2, kb)
		require.ErrorIs(err, ErrUnknownCodec)

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, c)
		err = viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) { return nil })
		require.ErrorIs(err, ErrUnknownCodec)
	})
	t.Run("Value builder must build value", func(t *testing.T) {
		vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))
		vb.PutInt64("id", 42)
		vb.PutString("name", "Coca Cola")
		vb.PutBool("active", true)

		v := vb.Build()

		require.Equal(int64(42), v.AsInt64("id"))
		require.Equal("Coca Cola", v.AsString("name"))
		require.Equal(true, v.AsBool("active"))
	})
	t.Run("Value builder must panic on build value", func(t *testing.T) {
		vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))
		vb.PutInt32("id", 42)

		require.Panics(func() { _ = vb.Build() })
	})
}

type entryType struct {
	wsid  istructs.WSID
	key   istructs.IKeyBuilder
	value istructs.IValueBuilder
}

func newEntry(viewRecords istructs.IViewRecords, wsid istructs.WSID, idDepartment int64, active bool, code string, id int64, name string) entryType {
	kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
	kb.PutInt64("partitionKey1", int64(wsid))
	kb.PutInt64("clusteringColumn1", idDepartment)
	kb.PutBool("clusteringColumn2", active)
	kb.PutString("clusteringColumn3", code)
	vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))
	vb.PutInt64("id", id)
	vb.PutString("name", name)
	vb.PutBool("active", active)
	return entryType{
		wsid:  wsid,
		key:   kb,
		value: vb,
	}
}

func Test_LoadStoreViewRecord_Bytes(t *testing.T) {
	require := require.New(t)

	viewName := istructs.NewQName("test", "view")

	schemas := schemas.NewSchemaCache()
	t.Run("must be ok to build schemas", func(t *testing.T) {
		schemas.AddView(viewName).
			AddPartField("pf_int32", istructs.DataKind_int32).
			AddPartField("pf_int64", istructs.DataKind_int64).
			AddPartField("pf_float32", istructs.DataKind_float32).
			AddPartField("pf_float64", istructs.DataKind_float64).
			AddPartField("pf_qname", istructs.DataKind_QName).
			AddPartField("pf_bool", istructs.DataKind_bool).
			AddPartField("pf_recID", istructs.DataKind_RecordID).
			AddClustColumn("cc_int32", istructs.DataKind_int32).
			AddClustColumn("cc_int64", istructs.DataKind_int64).
			AddClustColumn("cc_float32", istructs.DataKind_float32).
			AddClustColumn("cc_float64", istructs.DataKind_float64).
			AddClustColumn("cc_qname", istructs.DataKind_QName).
			AddClustColumn("cc_bool", istructs.DataKind_bool).
			AddClustColumn("cc_recID", istructs.DataKind_RecordID).
			AddClustColumn("cc_bytes", istructs.DataKind_bytes).
			AddValueField("vf_int32", istructs.DataKind_int32, true).
			AddValueField("vf_int64", istructs.DataKind_int64, false).
			AddValueField("vf_float32", istructs.DataKind_float32, false).
			AddValueField("vf_float64", istructs.DataKind_float64, false).
			AddValueField("vf_bytes", istructs.DataKind_bytes, false).
			AddValueField("vf_string", istructs.DataKind_string, false).
			AddValueField("vf_qname", istructs.DataKind_QName, false).
			AddValueField("vf_bool", istructs.DataKind_bool, false).
			AddValueField("vf_recID", istructs.DataKind_RecordID, false).
			AddValueField("vf_record", istructs.DataKind_Record, false).
			AddValueField("vf_event", istructs.DataKind_Event, false)
	})

	cfg := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app2, schemas)

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
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

	p, c := k1.storeToBytes()
	require.NotNil(p)
	require.NotNil(c)

	t.Run("should be success load", func(t *testing.T) {
		k2 := newKey(cfg, viewName)
		err := k2.loadFromBytes(p, c)
		require.NoError(err)

		testRowsIsEqual(t, &k1.partRow, &k2.partRow)
		testRowsIsEqual(t, &k1.clustRow, &k2.clustRow)

		require.True(k1.Equals(k2))
		require.True(k2.Equals(k1))

		k2.PutBytes("cc_bytes", []byte("TesT"))
		require.False(k1.Equals(k2))
	})

	t.Run("should be load error if truncated key bytes", func(t *testing.T) {
		k2 := newKey(cfg, viewName)
		for i := 0; i < len(p); i++ {
			err := k2.loadFromBytes(p[:i], c)
			require.Error(err, i)
		}
		for i := 0; i < len(p)-4; i++ { // 4 - is length of variable bytes "test" that can be truncated with impunity
			err := k2.loadFromBytes(p, c[:i])
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

	v, err := v1.storeToBytes()
	require.NoError(err)

	v2 := newValue(cfg, viewName)
	err = v2.loadFromBytes(v)
	require.NoError(err)

	testRowsIsEqual(t, &v1.rowType, &v2.rowType)
}

// Test_ViewRecords_ClustColumnsQName: see https://dev.heeus.io/launchpad/#!16377 problem
func Test_ViewRecords_ClustColumnsQName(t *testing.T) {
	require := require.New(t)
	ws := istructs.WSID(1234)

	// App schema, same as previous but with RecordID field in the clustering key
	//
	appConfigs := func() AppConfigsType {

		schemas := schemas.NewSchemaCache()
		t.Run("must be ok to build schemas", func(t *testing.T) {
			schemas.AddView(istructs.NewQName("test", "viewDrinks")).
				AddPartField("partitionKey1", istructs.DataKind_int64).
				AddClustColumn("clusteringColumn1", istructs.DataKind_QName).
				AddClustColumn("clusteringColumn2", istructs.DataKind_RecordID).
				AddValueField("id", istructs.DataKind_int64, true).
				AddValueField("name", istructs.DataKind_string, true).
				AddValueField("active", istructs.DataKind_bool, true)

			_ = schemas.Add(istructs.NewQName("test", "obj1"), istructs.SchemaKind_Object)
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		return cfgs
	}

	p := Provide(appConfigs(), iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
	as, err := p.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	viewRecords := as.ViewRecords()

	//
	// Add single record
	//
	kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
	kb.PutInt64("partitionKey1", int64(1))
	kb.PutQName("clusteringColumn1", istructs.NewQName("test", "obj1"))
	kb.PutRecordID("clusteringColumn2", 131072)
	vb := viewRecords.NewValueBuilder(istructs.NewQName("test", "viewDrinks"))
	vb.PutInt64("id", 123)
	vb.PutString("name", "Coca-cola")
	vb.PutBool("active", true)

	require.NoError(viewRecords.Put(ws, kb, vb))

	//
	// Fetch single record
	//
	t.Run("Test read single item", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(istructs.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", int64(1))
		kb.PutQName("clusteringColumn1", istructs.NewQName("test", "obj1"))
		kb.PutRecordID("clusteringColumn2", 131072)

		oldValue := istructs.IValue(nil)
		oldCcKey1 := istructs.NullQName
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
		require.Equal(istructs.NewQName("test", "obj1"), oldCcKey1)
		require.Equal(istructs.RecordID(131072), oldCcKey2)
	})
}

func Test_ViewRecord_GetBatch(t *testing.T) {
	require := require.New(t)

	championatsView := istructs.NewQName("test", "championats")
	championsView := istructs.NewQName("test", "champions")

	schemas := schemas.NewSchemaCache()
	t.Run("must be ok to build schemas", func(t *testing.T) {
		schemas.AddView(championatsView).
			AddPartField("Year", istructs.DataKind_int32).
			AddClustColumn("Sport", istructs.DataKind_string).
			AddValueField("Country", istructs.DataKind_string, true).
			AddValueField("City", istructs.DataKind_string, false)

		schemas.AddView(championsView).
			AddPartField("Year", istructs.DataKind_int32).
			AddClustColumn("Sport", istructs.DataKind_string).
			AddValueField("Winner", istructs.DataKind_string, true)
	})

	storage := teststore.NewTestStorage()
	storageProvider := teststore.NewTestStorageProvider(storage)

	cfgs := make(AppConfigsType, 1)
	_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)
	provider, _ := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

	app, err := provider.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)

	type championat struct {
		year              int32
		sport, cntr, city string
		winner            string
	}
	var championats = []championat{
		{1949, "Волейбол", "Чехословакия", "Прага", "СССР"},
		{1952, "Волейбол", "СССР", "Москва", "СССР"},
		{1956, "Волейбол", "Франция", "Париж", "Чехословакия"},
		{1960, "Волейбол", "Бразиия", "Рио-де-Жанейро", "СССР"},
		{1962, "Волейбол", "СССР", "Москва", "СССР"},
		{1966, "Волейбол", "Чехословакия", "Прага", "Чехословакия"},
		{1970, "Волейбол", "Болгария", "София", "ГДР"},
		{1974, "Волейбол", "Мексика", "Мехико", "Польша"},
		{1978, "Волейбол", "Италия", "Рим", "СССР"},
		{1982, "Волейбол", "Аргентина", "Буэнос-Айрес", "СССР"},
		{1986, "Волейбол", "Франция", "Париж", "США"},
		{1990, "Волейбол", "Бразиия", "Рио-де-Жанейро", "Италия"},
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
		{1995, "Гандбол", "Исландия", "Рейкявик", "Франция"},
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
		for _, c := range championats {
			kv := istructs.ViewKV{}
			kv.Key = app.ViewRecords().KeyBuilder(championatsView)
			kv.Key.PutInt32("Year", c.year)
			kv.Key.PutString("Sport", c.sport)
			kv.Value = app.ViewRecords().NewValueBuilder(championatsView)
			kv.Value.PutString("Country", c.cntr)
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
		for _, c := range championats {
			kv := istructs.ViewRecordGetBatchItem{}
			kv.Key = app.ViewRecords().KeyBuilder(championatsView)
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

		err := app.ViewRecords().(*appViewRecordsType).GetBatch(1, batch)
		require.NoError(err)

		i := 0
		for _, c := range championats {
			b := batch[i]
			require.True(b.Ok)
			require.Equal(c.cntr, b.Value.AsString("Country"))
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

		err := app.ViewRecords().(*appViewRecordsType).GetBatch(1, batch)
		require.NoError(err)

		require.True(batch[0].Ok)
		require.Equal("СССР", batch[0].Value.AsString("Winner"))

		require.True(batch[1].Ok)
		require.Equal("Россия", batch[1].Value.AsString("Winner"))

		require.False(batch[2].Ok)
		require.Equal(istructs.NullQName, batch[2].Value.AsQName(istructs.SystemField_QName))
	})

	t.Run("must fail to read if maximum batch size exceeds", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, maxGetBatchRecordCount+1)
		for i := 0; i < len(batch); i++ {
			batch[i].Key = app.ViewRecords().KeyBuilder(championsView)
			batch[i].Key.PutInt32("Year", int32(i))
			batch[i].Key.PutString("Sport", "Шашки")
		}
		err := app.ViewRecords().(*appViewRecordsType).GetBatch(1, batch)
		require.ErrorIs(err, ErrMaxGetBatchRecordCountExceeds)
	})

	t.Run("must fail to read if some key build error", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 3)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt64("Year", 1962) // error here
		batch[0].Key.PutString("Sport", "Волейбол")

		err := app.ViewRecords().(*appViewRecordsType).GetBatch(1, batch)
		require.ErrorIs(err, ErrWrongFieldType)
	})

	t.Run("must fail to read if some key is not valid", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 3)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		//batch[0].Key.PutString("Sport", "Волейбол") // error here

		err := app.ViewRecords().(*appViewRecordsType).GetBatch(1, batch)
		require.ErrorIs(err, ErrFieldIsEmpty)
	})

	t.Run("must fail to read if storage GetBatch failed", func(t *testing.T) {
		testError := errors.New("test error")

		batch := make([]istructs.ViewRecordGetBatchItem, 1)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		batch[0].Key.PutString("Sport", "Волейбол")

		storage.ScheduleGetError(testError, nil, []byte("Волейбол")) // error here

		err := app.ViewRecords().(*appViewRecordsType).GetBatch(1, batch)
		require.ErrorIs(err, testError)
	})

	t.Run("must fail to read if storage GetBatch returns damaged data", func(t *testing.T) {
		batch := make([]istructs.ViewRecordGetBatchItem, 1)
		batch[0].Key = app.ViewRecords().KeyBuilder(championsView)
		batch[0].Key.PutInt32("Year", 1962)
		batch[0].Key.PutString("Sport", "Волейбол")

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, []byte("Волейбол"))

		err := app.ViewRecords().(*appViewRecordsType).GetBatch(1, batch)
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

		k4 := app.ViewRecords().KeyBuilder(championatsView)
		k4.PutInt32("Year", 1962)
		k4.PutString("Sport", "Волейбол")

		require.False(k1.Equals(k4), "KeyBuilder must not be equals if different QNames")
	})
}
