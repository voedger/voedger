/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef/builder"
	log "github.com/voedger/voedger/pkg/goutils/logger"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func Test_newRecord(t *testing.T) {

	require := require.New(t)
	test := test()

	t.Run("newNullRecord must return empty, nullQName record with specified sys.ID", func(t *testing.T) {
		rec := NewNullRecord(100500)
		require.Equal(appdef.NullQName, rec.QName())
		require.Equal(istructs.RecordID(100500), rec.ID())
		require.Equal(istructs.NullRecordID, rec.Parent())
		require.Equal("", rec.Container())
	})

	t.Run("newRecord must return empty, nullQName record", func(t *testing.T) {
		rec := newRecord(test.AppCfg)
		require.True(rec.empty())

		t.Run("test as IRecord", func(t *testing.T) {
			var r istructs.IRecord = rec
			require.Equal(appdef.NullQName, r.QName())
			require.Equal(istructs.NullRecordID, r.ID())
			require.Equal(istructs.NullRecordID, r.Parent())
			require.Equal("", r.Container())
		})

		t.Run("test as ICRecord", func(t *testing.T) {
			var r istructs.ICRecord = rec
			require.True(r.IsActive())
		})

		t.Run("test as IRowReader", func(t *testing.T) {
			var r istructs.IRowReader = rec
			require.Equal(appdef.NullQName, r.AsQName(appdef.SystemField_QName))
			require.Equal(istructs.NullRecordID, r.AsRecordID(appdef.SystemField_ID))
			require.Equal(istructs.NullRecordID, r.AsRecordID(appdef.SystemField_ParentID))
			require.Equal("", r.AsString(appdef.SystemField_Container))
			require.True(r.AsBool(appdef.SystemField_IsActive))
		})

		t.Run("test as ICUDRow", func(t *testing.T) {
			var r istructs.ICUDRow = rec
			require.False(r.IsActivated())
			require.False(r.IsDeactivated())
			require.False(r.IsNew())
		})
	})

	t.Run("newEmptyTestCDoc must return empty, «test.CDoc»", func(t *testing.T) {
		doc := newEmptyTestCDoc()
		require.True(doc.empty())
		require.Equal(test.testCDoc, doc.QName())
		require.Equal(istructs.NullRecordID, doc.ID())
		require.True(doc.IsActive())

		t.Run("newEmptyTestCRec must return empty, «test.Record»", func(t *testing.T) {
			rec := newEmptyTestCRecord()
			require.True(rec.empty())
			require.Equal(test.testCRec, rec.QName())
			require.Equal(istructs.NullRecordID, rec.ID())
			require.Equal(istructs.NullRecordID, rec.Parent())
			require.Equal("", rec.Container())
			require.True(rec.IsActive())
		})
	})

	t.Run("newTestCDoc must return non empty, full filled and valid «test.CDoc»", func(t *testing.T) {
		doc := newTestCDoc(100500)
		require.False(doc.empty())
		require.Equal(test.testCDoc, doc.QName())
		require.Equal(istructs.RecordID(100500), doc.ID())
		require.Equal(istructs.RecordID(100500), doc.AsRecordID(appdef.SystemField_ID))
		require.Equal(istructs.NullRecordID, doc.Parent())
		require.Equal("", doc.Container())
		require.True(doc.IsActive())

		testTestCDoc(t, doc, 100500)

		t.Run("system field counters for test CDoc", func(t *testing.T) {
			sysCnt := 0
			for _, f := range doc.fields.Fields() {
				require.True(doc.HasValue(f.Name()))
				if f.IsSys() {
					sysCnt++
				}
			}
			require.Equal(3, sysCnt) // sys.QName, sys.ID and sys.IsActive
		})

		t.Run("inactivating test CDoc", func(t *testing.T) {
			doc.PutBool(appdef.SystemField_IsActive, false)

			require.False(doc.IsActive())
			require.False(doc.AsBool(appdef.SystemField_IsActive))
		})

		t.Run("field counters for test CDoc", func(t *testing.T) {
			cnt := 0
			sysCnt := 0

			for _, f := range doc.fields.Fields() {
				require.True(doc.HasValue(f.Name()))
				if f.IsSys() {
					sysCnt++
				}
				cnt++
			}

			require.Equal(3, sysCnt) // sys.QName, sys.ID and sys.IsActive
			require.Equal(sysCnt+11, cnt)
			require.Equal(doc.fields.FieldCount(), cnt)
		})

		t.Run("newTestCRec must return non empty, full filled and valid «test.Record»", func(t *testing.T) {
			const recID istructs.RecordID = 100501
			rec := newTestCRecord(recID)
			require.False(rec.empty())
			require.Equal(test.testCRec, rec.QName())
			require.Equal(recID, rec.ID())
			require.Equal(recID, rec.AsRecordID(appdef.SystemField_ID))
			require.Equal(istructs.NullRecordID, rec.Parent())
			require.Equal("", rec.Container())
			require.True(rec.IsActive())

			testTestCRec(t, rec, recID)

			rec.PutRecordID(appdef.SystemField_ParentID, doc.ID())
			require.Equal(doc.ID(), rec.Parent())
			require.Equal(doc.ID(), rec.AsRecordID(appdef.SystemField_ParentID))

			rec.PutString(appdef.SystemField_Container, "record")
			require.Equal("record", rec.Container())
			require.Equal("record", rec.AsString(appdef.SystemField_Container))

			t.Run("Should be ok to get sys.ID and sys.ParentID through AsInt64() and AsFloat64()", func(t *testing.T) {
				require.Equal(float64(recID), rec.AsFloat64(appdef.SystemField_ID))
				require.Equal(float64(doc.ID()), rec.AsFloat64(appdef.SystemField_ParentID))
				require.Equal(int64(recID), rec.AsInt64(appdef.SystemField_ID))
				require.Equal(int64(doc.ID()), rec.AsInt64(appdef.SystemField_ParentID))
			})

			t.Run("system field counters for test CRecord", func(t *testing.T) {
				sysCnt := 0

				for _, f := range rec.fields.Fields() {
					require.True(rec.HasValue(f.Name()))
					if f.IsSys() {
						sysCnt++
					}
				}

				require.Equal(5, sysCnt) // sys.QName, sys.ID sys.ParentID, sys.Container and sys.IsActive
			})

			t.Run("inactivating test CRecord", func(t *testing.T) {
				rec.PutBool(appdef.SystemField_IsActive, false)

				require.False(rec.IsActive())
				require.False(rec.AsBool(appdef.SystemField_IsActive))
			})

			t.Run("field counters for test CRecord", func(t *testing.T) {
				cnt := 0
				sysCnt := 0

				for _, f := range rec.fields.Fields() {
					require.True(rec.HasValue(f.Name()))
					if f.IsSys() {
						sysCnt++
					}
					cnt++
				}

				require.Equal(5, sysCnt) // sys.QName, sys.ID sys.ParentID, sys.Container and sys.IsActive
				require.Equal(sysCnt+11, cnt)
				require.Equal(rec.fields.FieldCount(), cnt)
			})
		})
	})
}

func Test_LoadStoreRecord_Bytes(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("test rec1 must be success storeToBytes() and test rec2 must success loadFromBytes(). rec1 and rec2 must be equals", func(t *testing.T) {
		rec1 := newTestCDoc(100500)

		b := rec1.storeToBytes()

		rec2 := newRecord(test.AppCfg)
		err := rec2.loadFromBytes(b)
		require.NoError(err)
		testTestCDoc(t, rec2, 100500)

		testRecsIsEqual(t, rec1, rec2)
	})

	t.Run("same as previous test, but for deactivated CDoc", func(t *testing.T) {
		rec1 := newTestCDoc(100501)
		rec1.PutBool(appdef.SystemField_IsActive, false)

		b := rec1.storeToBytes()

		rec2 := newRecord(test.AppCfg)
		err := rec2.loadFromBytes(b)
		require.NoError(err)
		testTestCDoc(t, rec2, 100501)
		require.False(rec2.AsBool(appdef.SystemField_IsActive))

		testRecsIsEqual(t, rec1, rec2)
	})

	t.Run("should be ok to read data stored with previous codec versions", func(t *testing.T) {
		store_codec_RawDynoBuffer := func(row *recordType) (out []byte) {
			buf := new(bytes.Buffer)
			_ = binary.Write(buf, binary.BigEndian, codec_RawDynoBuffer)
			id, err := row.qNameID()
			require.NoError(err)
			_ = binary.Write(buf, binary.BigEndian, int16(id))
			if row.QName() == appdef.NullQName {
				return buf.Bytes()
			}
			if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_ID); exists {
				require.NoError(binary.Write(buf, binary.BigEndian, uint64(row.ID())))
			}
			if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_ParentID); exists {
				require.NoError(binary.Write(buf, binary.BigEndian, uint64(row.parentID)))
			}
			if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_Container); exists {
				id, _ := row.containerID()
				require.NoError(binary.Write(buf, binary.BigEndian, int16(id)))
			}
			if exists, _ := row.typ.Kind().HasSystemField(appdef.SystemField_IsActive); exists {
				require.NoError(binary.Write(buf, binary.BigEndian, row.isActive))
			}
			b, err := row.dyB.ToBytes()
			require.NoError(err)
			length := uint32(len(b))
			require.NoError(binary.Write(buf, binary.BigEndian, &length))
			_, err = buf.Write(b)
			require.NoError(err)
			return buf.Bytes()
		}

		t.Run("test CDocs", func(t *testing.T) {
			doc1 := newTestCDoc(100502)

			bytes := store_codec_RawDynoBuffer(doc1)

			doc2 := newRecord(test.AppCfg)
			err := doc2.loadFromBytes(bytes)
			require.NoError(err)
			testTestCDoc(t, doc2, 100502)

			testRecsIsEqual(t, doc1, doc2)
		})

		t.Run("test CRecords", func(t *testing.T) {
			rec1 := newTestCRecord(100503)
			rec1.PutRecordID(appdef.SystemField_ParentID, 100502)
			rec1.PutString(appdef.SystemField_Container, test.goodIdent)

			bytes := store_codec_RawDynoBuffer(rec1)

			rec2 := newRecord(test.AppCfg)
			err := rec2.loadFromBytes(bytes)
			require.NoError(err)

			testRecsIsEqual(t, rec1, rec2)
		})
	})

	t.Run("null records (with NullQName) must be success storeToBytes() and success loadFromBytes()", func(t *testing.T) {
		rec1 := newRecord(test.AppCfg)
		b := rec1.storeToBytes()

		rec2 := newEmptyTestCDoc()
		err := rec2.loadFromBytes(b)
		require.NoError(err)

		require.Equal(appdef.NullQName, rec2.QName())
		require.Equal(istructs.NullRecordID, rec2.ID())
	})

	t.Run("empty records (with «test.record» QName) must be success storeToBytes() and success loadFromBytes()", func(t *testing.T) {
		rec1 := newEmptyTestCDoc()
		b := rec1.storeToBytes()

		rec2 := newRecord(test.AppCfg)
		err := rec2.loadFromBytes(b)
		require.NoError(err)

		require.Equal(test.testCDoc, rec2.QName())
		require.Equal(istructs.NullRecordID, rec2.ID())
	})

	t.Run("test rec1 must be success storeToBytes(); rec2 loadFromBytes() from truncated bytes must fails", func(t *testing.T) {
		rec1 := newTestCDoc(100500)

		b := rec1.storeToBytes()

		length := len(b)
		for i := 0; i < length; i++ {
			corrupted := b[0:i]
			rec2 := newRecord(test.AppCfg)
			err := rec2.loadFromBytes(corrupted)
			require.Error(err, fmt.Sprintf("unexpected success load record from bytes truncated at length «%d»", i))
		}
	})

	t.Run("dynobuffer corrupt test: loadFromBytes() from corrupted bytes may:\n"+
		"— fail (Panic or Error) or\n"+
		"— success read wrong data (BadData) or\n"+
		"— success read correct data (Lucky)",
		func(t *testing.T) {
			if testing.Short() {
				t.Skip()
			}
			rec1 := newTestCDoc(100500)

			b := rec1.storeToBytes()

			length := len(b)
			stat := make(map[string]int)
			for i := 0; i < length; i++ {
				b[i] ^= 255
				rec2 := newRecord(test.AppCfg)
				func() {
					defer func() {
						if err := recover(); err != nil {
							log.Verbose("%d: panic at read record: %v", i, err)
							stat["Panics"]++
						}
					}()
					if err := rec2.loadFromBytes(b); err != nil {
						log.Verbose("%d: error at load: %v\n", i, err)
						stat["Errors"]++
						return
					}
					if ok, diff := recsIsEqual(rec1, rec2); ok {
						log.Verbose("%d: success load, data is ok\n", i)
						stat["Lucky"]++
					} else {
						log.Verbose("%d: success load, data is corrupted: %v\n", i, diff)
						stat["BadData"]++
					}
				}()
				b[i] ^= 255
			}
			log.Verbose("len: %d, stat: %v\n", length, stat)
		})

	t.Run("test field renaming availability", func(t *testing.T) {
		rec1 := newTestCDoc(100500)

		b := rec1.storeToBytes()

		newFieldName := func(oldValue string) string { return oldValue + "_1" }
		oldFieldName := func(newValue string) string { return newValue[:len(newValue)-2] }

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		t.Run("should be ok to build new ver of application", func(t *testing.T) {
			wsb := adb.AddWorkspace(testData.wsName)
			newCDoc := wsb.AddCDoc(test.testCDoc)

			oldCDoc := appdef.CDoc(rec1.appCfg.AppDef.Type, test.testCDoc)
			for _, f := range oldCDoc.Fields() {
				if !f.IsSys() {
					newCDoc.AddField(newFieldName(f.Name()), f.DataKind(), f.Required())
				}
			}
			wsb.AddObject(test.tablePhotos) // for reading QName_1 field value
		})

		newConfig := newBuiltInAppConfig(test.AppCfg.Name, adb)
		newConfig.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		err := newConfig.prepare(nil, test.AppCfg.storage)
		require.NoError(err)

		rec2 := newRecord(newConfig)
		err = rec2.loadFromBytes(b)
		require.NoError(err)

		require.Equal(rec1.QName(), rec2.QName())
		rec1.dyB.IterateFields(nil, func(name string, val1 interface{}) bool {
			newName := name
			if !appdef.IsSysField(name) {
				newName = newFieldName(name)
			}
			require.True(rec2.HasValue(newName), newName)
			val2 := rec2.dyB.Get(newName)
			require.Equal(val1, val2)
			return true
		})
		rec2.dyB.IterateFields(nil, func(name string, val2 interface{}) bool {
			oldName := name
			if !appdef.IsSysField(name) {
				oldName = oldFieldName(name)
			}
			require.True(rec1.HasValue(oldName), oldName)
			return true
		})
	})

}

func Test_fieldValue(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("should return correct field values", func(t *testing.T) {

		t.Run(fmt.Sprint(test.testCDoc), func(t *testing.T) {
			rec := newTestCDoc(100500)
			tests := []struct {
				f appdef.FieldName
				v interface{}
			}{
				{appdef.SystemField_QName, test.testCDoc},
				{appdef.SystemField_ID, istructs.RecordID(100500)},
				{appdef.SystemField_IsActive, true},
				{"int32", int32(1)},
				{"int64", int64(2)},
				{"float32", float32(3)},
				{"float64", float64(4)},
				{"bytes", []byte{1, 2, 3, 4, 5}},
				{"string", "test string"},
				{"raw", test.photoRawValue},
				{"QName", test.tablePhotos},
				{"bool", true},
				{"RecordID", istructs.RecordID(7777777)},
			}

			for _, test := range tests {
				f := rec.fieldDef(test.f)
				require.NotNil(f, test.f)
				require.Equal(test.v, rec.fieldValue(f))
			}
		})

		t.Run(fmt.Sprint(test.testViewRecord.name), func(t *testing.T) {
			vv := newTestViewValue()
			tests := []struct {
				f appdef.FieldName
				v interface{}
			}{
				{appdef.SystemField_QName, test.testViewRecord.name},
				{test.testViewRecord.valueFields.buyer, test.buyerValue},
				{test.testViewRecord.valueFields.age, test.ageValue},
				{test.testViewRecord.valueFields.heights, test.heightValue},
				{test.testViewRecord.valueFields.human, true},
				{test.testViewRecord.valueFields.photo, test.photoValue},
			}

			for _, test := range tests {
				f := vv.fieldDef(test.f)
				require.NotNil(f, test.f)
				require.Equal(test.v, vv.fieldValue(f), test.f)
			}

			t.Run("complex fields", func(t *testing.T) {

				t.Run(test.testViewRecord.valueFields.record, func(t *testing.T) {
					val := vv.fieldValue(vv.fieldDef(test.testViewRecord.valueFields.record))
					require.NotNil(val)
					rec, ok := val.(istructs.IRecord)
					require.True(ok)
					testTestCDoc(t, rec, 100888)
				})

				t.Run(test.testViewRecord.valueFields.event, func(t *testing.T) {
					val := vv.fieldValue(vv.fieldDef(test.testViewRecord.valueFields.event))
					require.NotNil(val)
					ev, ok := val.(istructs.IDbEvent)
					require.True(ok)
					testTestEvent(t, ev, 100500, 1050, true)
				})
			})
		})
	})
}

func TestSpecifiedValues(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("should has no modifications if new record", func(t *testing.T) {
		rec := newRecord(test.AppCfg)
		for range rec.SpecifiedValues {
			t.Fail()
		}
	})

	testEnum := func(rec istructs.ICUDRow, want map[appdef.FieldName]interface{}) {
		t.Run("enum", func(t *testing.T) {
			got := make(map[appdef.FieldName]interface{})
			for n, v := range rec.SpecifiedValues {
				got[n.Name()] = v
			}
			require.Equal(want, got)
		})

		t.Run("breakable", func(t *testing.T) {
			for stop := range want {
				cnt := 0
				for n := range rec.SpecifiedValues {
					if n.Name() == stop {
						break
					}
					cnt++
				}
				require.Less(cnt, len(want))
			}
		})
	}

	t.Run("should enum modified user fields", func(t *testing.T) {
		rec := newRecord(test.AppCfg)
		rec.setQName(test.testCDoc)
		rec.PutInt32("int32", 1)
		rec.PutString("string", "test")
		rec.PutInt64("int64", 0)   // zero (0) value should be enumerated too
		rec.PutBool("bool", false) // zero (false) value should be enumerated too
		rec.PutRecordID("RecordID", 3)
		rec.PutInt64("RecordID_2", 4)
		rec.PutQName("QName", qNameMyQName)
		require.NoError(rec.build())

		testEnum(rec,
			map[appdef.FieldName]interface{}{
				"int32":                     int32(1),
				"string":                    "test",
				"int64":                     int64(0),
				"bool":                      false,
				appdef.SystemField_QName:    test.testCDoc,
				appdef.SystemField_IsActive: true,
				"RecordID":                  istructs.RecordID(3),
				"RecordID_2":                istructs.RecordID(4),
				"QName":                     qNameMyQName,
			})
	})

	t.Run("should enum modified sys.IsActive", func(t *testing.T) {
		rec := newRecord(test.AppCfg)
		rec.setQName(test.testCDoc)
		rec.PutBool(appdef.SystemField_IsActive, false) // system field should be enumerated
		require.NoError(rec.build())

		testEnum(rec,
			map[appdef.FieldName]interface{}{
				appdef.SystemField_IsActive: false,
				appdef.SystemField_QName:    test.testCDoc,
			})
	})

	// #2785
	t.Run("should enum emptied user fields", func(t *testing.T) {
		rec := newRecord(test.AppCfg)
		rec.setQName(test.testCDoc)
		rec.PutBytes("bytes", []byte{})
		rec.PutString("string", "")
		rec.PutBytes("raw", nil)
		require.NoError(rec.build())

		testEnum(rec,
			map[appdef.FieldName]interface{}{
				"bytes":                     []byte{},
				"string":                    "",
				"raw":                       []byte{},
				appdef.SystemField_QName:    test.testCDoc,
				appdef.SystemField_IsActive: true,
			})
	})
}
