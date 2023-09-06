/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
)

func Test_rowNullDefinition(t *testing.T) {
	require := require.New(t)

	row := newTestRow()

	row.setQName(appdef.NullQName)
	require.Equal(appdef.NullQName, row.QName())

	row.setDef(appdef.NullDef)
	require.Equal(appdef.NullQName, row.QName())

	row.setDef(nil)
	require.Equal(appdef.NullQName, row.QName())
}

func Test_dynoBufValue(t *testing.T) {
	require := require.New(t)
	test := test()

	row := newTestRow()

	t.Run("test int32", func(t *testing.T) {
		v, err := row.dynoBufValue(int32(7), appdef.DataKind_int32)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue(float64(7), appdef.DataKind_int32)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue("7", appdef.DataKind_int32)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test int64", func(t *testing.T) {
		v, err := row.dynoBufValue(int64(7), appdef.DataKind_int64)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue(float64(7), appdef.DataKind_int64)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue("7", appdef.DataKind_int64)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test float32", func(t *testing.T) {
		v, err := row.dynoBufValue(float32(7.7), appdef.DataKind_float32)
		require.NoError(err)
		require.EqualValues(7.7, v)

		v, err = row.dynoBufValue(float64(7.7), appdef.DataKind_float32)
		require.NoError(err)
		require.EqualValues(7.7, v)

		v, err = row.dynoBufValue("7.7", appdef.DataKind_float32)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test float64", func(t *testing.T) {
		v, err := row.dynoBufValue(float64(7.7), appdef.DataKind_float64)
		require.NoError(err)
		require.EqualValues(7.7, v)

		v, err = row.dynoBufValue(7, appdef.DataKind_float64)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test []byte", func(t *testing.T) {
		v, err := row.dynoBufValue([]byte{1, 2, 3}, appdef.DataKind_bytes)
		require.NoError(err)
		require.Equal([]byte{1, 2, 3}, v)

		// cspell:disable
		v, err = row.dynoBufValue("AQIDBA==", appdef.DataKind_bytes)
		// cspell:enable
		require.NoError(err)
		require.Equal([]byte{1, 2, 3, 4}, v)

		v, err = row.dynoBufValue("ups", appdef.DataKind_bytes)
		require.Error(err) // base64 convert error
		require.Nil(v)

		v, err = row.dynoBufValue(7, appdef.DataKind_bytes)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test QName", func(t *testing.T) {
		id, _ := test.AppCfg.qNames.ID(test.saleCmdName)
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(id))

		v, err := row.dynoBufValue(test.saleCmdName, appdef.DataKind_QName)
		require.NoError(err)
		require.EqualValues(b, v)

		v, err = row.dynoBufValue(test.saleCmdName.String(), appdef.DataKind_QName)
		require.NoError(err)
		require.EqualValues(b, v)

		v, err = row.dynoBufValue(appdef.NewQName("test", "unknown"), appdef.DataKind_QName)
		require.ErrorIs(err, qnames.ErrNameNotFound)
		require.Nil(v)

		v, err = row.dynoBufValue("test.unknown", appdef.DataKind_QName)
		require.ErrorIs(err, qnames.ErrNameNotFound)
		require.Nil(v)

		v, err = row.dynoBufValue("ups!", appdef.DataKind_QName)
		require.Error(err) // invalid QName
		require.Nil(v)

		v, err = row.dynoBufValue(7, appdef.DataKind_bytes)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test bool", func(t *testing.T) {
		v, err := row.dynoBufValue(false, appdef.DataKind_bool)
		require.NoError(err)
		require.Equal(false, v)

		v, err = row.dynoBufValue(true, appdef.DataKind_bool)
		require.NoError(err)
		require.Equal(true, v)

		v, err = row.dynoBufValue(7, appdef.DataKind_bool)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test int64", func(t *testing.T) {
		v, err := row.dynoBufValue(istructs.NullRecordID, appdef.DataKind_RecordID)
		require.NoError(err)
		require.EqualValues(istructs.NullRecordID, v)

		v, err = row.dynoBufValue(istructs.RecordID(100500700), appdef.DataKind_RecordID)
		require.NoError(err)
		require.EqualValues(100500700, v)

		v, err = row.dynoBufValue(float64(100500700), appdef.DataKind_RecordID)
		require.NoError(err)
		require.EqualValues(100500700, v)

		v, err = row.dynoBufValue("100500700", appdef.DataKind_RecordID)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test Record", func(t *testing.T) {
		var testRec istructs.IRecord = newTestCRecord(100500700)

		checkRecord := func(data interface{}) {
			b, ok := data.([]byte)
			require.True(ok)
			require.NotNil(b)
			r := newTestCRecord(istructs.NullRecordID)
			err := r.loadFromBytes(b)
			require.NoError(err)
			testRecsIsEqual(t, testRec, r)
		}

		v, err := row.dynoBufValue(testRec, appdef.DataKind_Record)
		require.NoError(err)
		checkRecord(v)

		v, err = row.dynoBufValue("ups", appdef.DataKind_Record)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test Event", func(t *testing.T) {
		var testEvent istructs.IDbEvent = newTestEvent(100501, 100500700)

		checkEvent := func(data interface{}) {
			b, ok := data.([]byte)
			require.True(ok)
			require.NotNil(b)
			e := newEmptyTestEvent()
			err := e.loadFromBytes(b)
			require.NoError(err)
			testTestEvent(t, testEvent, 100501, 100500700, false)
		}

		v, err := row.dynoBufValue(testEvent, appdef.DataKind_Event)
		require.NoError(err)
		checkEvent(v)

		v, err = row.dynoBufValue("ups", appdef.DataKind_Event)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})
}

func Test_rowType_PutAs_SimpleTypes(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("Put××× and As××× row methods for simple types", func(t *testing.T) {
		row1 := newTestRow()

		testTestRow(t, row1)

		row2 := newRow(nil)
		row2.copyFrom(row1)

		testRowsIsEqual(t, row1, &row2)

		testTestRow(t, &row2)
	})

	t.Run("As××× row methods must return default values if not calls Put×××", func(t *testing.T) {
		row := newEmptyTestRow()

		require.Equal(int32(0), row.AsInt32("int32"))
		require.Equal(int64(0), row.AsInt64("int64"))
		require.Equal(float32(0), row.AsFloat32("float32"))
		require.Equal(float64(0), row.AsFloat64("float64"))
		require.Equal([]byte(nil), row.AsBytes("bytes"))
		require.Equal("", row.AsString("string"))
		require.Equal(appdef.NullQName, row.AsQName("QName"))
		require.Equal(false, row.AsBool("bool"))
		require.Equal(istructs.NullRecordID, row.AsRecordID("RecordID"))

		val := newEmptyViewValue()
		require.Equal(istructs.IDbEvent(nil), val.AsEvent(test.testViewRecord.valueFields.event))
		rec := val.AsRecord(test.testViewRecord.valueFields.record)
		require.Equal(appdef.NullQName, rec.QName())
	})

	t.Run("PutNumber to numeric-type fields must be available (json)", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutNumber("int32", 1)
		row.PutNumber("int64", 2)
		row.PutNumber("float32", 3)
		row.PutNumber("float64", 4)
		row.PutNumber("RecordID", 5)

		require.NoError(row.build())

		require.Equal(int32(1), row.AsInt32("int32"))
		require.Equal(int64(2), row.AsInt64("int64"))
		require.Equal(float32(3), row.AsFloat32("float32"))
		require.Equal(float64(4), row.AsFloat64("float64"))
		require.Equal(istructs.RecordID(5), row.AsRecordID("RecordID"))
	})

	t.Run("PutChars to char-type fields (string, bytes and QName) must be available (json)", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutChars("string", "test 🏐 тест")
		row.PutChars("QName", test.saleCmdName.String())
		// cspell:disable
		row.PutChars("bytes", "AQIDBA==")
		// cspell:enable

		require.NoError(row.build())

		require.Equal("test 🏐 тест", row.AsString("string"))
		require.Equal(test.saleCmdName, row.AsQName("QName"))
		require.Equal([]byte{1, 2, 3, 4}, row.AsBytes("bytes"))
	})
}

func Test_rowType_PutAs_ComplexTypes(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("PutRecord and PutEvent / AsRecord and AsEvent row methods (via IValue)", func(t *testing.T) {
		row1 := newTestViewValue()
		testTestViewValue(t, row1)

		row2 := newRow(test.AppCfg)
		row2.copyFrom(row1)
		testTestViewValue(t, &row2)

		testRowsIsEqual(t, row1, &row2)
	})

	t.Run("must success NullRecord value for PutRecord / AsRecord methods", func(t *testing.T) {
		row := newEmptyViewValue()
		row.PutString(test.testViewRecord.valueFields.buyer, "buyer")
		row.PutRecord(test.testViewRecord.valueFields.record, NewNullRecord(istructs.NullRecordID))
		require.NoError(row.build())

		rec := row.AsRecord(test.testViewRecord.valueFields.record)
		require.NotNil(rec)
		require.Equal(appdef.NullQName, rec.QName())
		require.Equal(istructs.NullRecordID, rec.AsRecordID(appdef.SystemField_ID))
	})
}

func Test_rowType_PutErrors(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("Put××× to unknown field names must be error", func(t *testing.T) {

		testPut := func(put func(row istructs.IRowWriter)) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)
			put(&row)
			require.ErrorIs(row.build(), ErrNameNotFound)
		}

		testPut(func(row istructs.IRowWriter) { row.PutInt32("unknown_field", 1) })
		testPut(func(row istructs.IRowWriter) { row.PutInt32("unknown_field", 1) })
		testPut(func(row istructs.IRowWriter) { row.PutInt64("unknown_field", 2) })
		testPut(func(row istructs.IRowWriter) { row.PutFloat32("unknown_field", 3) })
		testPut(func(row istructs.IRowWriter) { row.PutFloat64("unknown_field", 4) })
		testPut(func(row istructs.IRowWriter) { row.PutBytes("unknown_field", []byte{1, 2, 3}) })
		testPut(func(row istructs.IRowWriter) { row.PutString("unknown_field", "abc") })
		testPut(func(row istructs.IRowWriter) { row.PutQName("unknown_field", istructs.QNameForError) })
		testPut(func(row istructs.IRowWriter) { row.PutBool("unknown_field", true) })
		testPut(func(row istructs.IRowWriter) { row.PutRecordID("unknown_field", istructs.NullRecordID) })

		testPut(func(row istructs.IRowWriter) { row.PutNumber("unknown_field", 888) })
		testPut(func(row istructs.IRowWriter) { row.PutChars("unknown_field", "c.h.a.r.s.") })
	})

	t.Run("Put××× with wrong types must be error", func(t *testing.T) {

		testPut := func(put func(row istructs.IRowWriter)) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)
			put(&row)
			require.ErrorIs(row.build(), ErrWrongFieldType)
		}

		testPut(func(row istructs.IRowWriter) { row.PutInt32("int64", 1) })
		testPut(func(row istructs.IRowWriter) { row.PutInt64("float32", 2) })
		testPut(func(row istructs.IRowWriter) { row.PutFloat32("int32", 3) })
		testPut(func(row istructs.IRowWriter) { row.PutFloat64("string", 4) })
		testPut(func(row istructs.IRowWriter) { row.PutBytes("float64", []byte{1, 2, 3}) })
		testPut(func(row istructs.IRowWriter) { row.PutString("bytes", "abc") })
		testPut(func(row istructs.IRowWriter) { row.PutQName("RecordID", istructs.QNameForError) })
		testPut(func(row istructs.IRowWriter) { row.PutBool("QName", true) })
		testPut(func(row istructs.IRowWriter) { row.PutRecordID("bool", istructs.NullRecordID) })
	})

	t.Run("PutNumber to non-numeric type field must be error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutNumber("bytes", 29)

		require.ErrorIs(row.build(), ErrWrongFieldType)
	})

	t.Run("PutQName with unknown QName value must be error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutQName("QName", appdef.NewQName("unknown", "unknown"))

		require.ErrorIs(row.build(), qnames.ErrNameNotFound)
	})

	t.Run("PutChars error handling", func(t *testing.T) {
		t.Run("PutChars to non-char type fields must be error", func(t *testing.T) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)

			row.PutChars("int32", "29")

			require.ErrorIs(row.build(), ErrWrongFieldType)
		})

		t.Run("PutChars to QName-type fields non convertible value must be error", func(t *testing.T) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)

			row.PutChars("QName", "welcome.2.error")

			require.ErrorIs(row.build(), appdef.ErrInvalidQNameStringRepresentation)
		})

		t.Run("PutChars to bytes-type fields non convertible base64 value must be error", func(t *testing.T) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)

			row.PutChars("bytes", "welcome.2.error")

			require.ErrorContains(row.build(), "illegal base64 data")
		})
	})

	t.Run("Multiply Put××× errors must be concatenated in build error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutFloat32("unknown_field", 555.5)
		row.PutInt32("int64", 1)

		err := row.build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorIs(err, ErrWrongFieldType)
	})

	t.Run("Must be error to put into abstract table", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.abstractDef)

		row.PutInt32("int32", 1)

		err := row.build()
		require.ErrorIs(err, ErrAbstractDefinition)
	})
}

func Test_rowType_AsPanics(t *testing.T) {
	t.Run("As××× unknown fields must panic", func(t *testing.T) {
		require := require.New(t)
		row := newTestRow()

		require.Panics(func() { row.AsInt32("unknownField") })
		require.Panics(func() { row.AsInt64("unknownField") })
		require.Panics(func() { row.AsFloat32("unknownField") })
		require.Panics(func() { row.AsFloat64("unknownField") })
		require.Panics(func() { row.AsBytes("unknownField") })
		require.Panics(func() { row.AsString("unknownField") })
		require.Panics(func() { row.AsQName("unknownField") })
		require.Panics(func() { row.AsBool("unknownField") })
		require.Panics(func() { row.AsRecordID("unknownField") })
		require.Panics(func() { row.AsRecord("unknownField") })
		require.Panics(func() { row.AsEvent("unknownField") })
	})
}

func Test_rowType_RecordIDs(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("RecordIDs must iterate all IDs", func(t *testing.T) {

		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutRecordID("RecordID", 1)
		row.PutRecordID("RecordID_2", 2)

		require.NoError(row.build())

		cnt := 0
		row.RecordIDs(true,
			func(name string, value istructs.RecordID) {
				switch name {
				case "RecordID":
					require.Equal(istructs.RecordID(1), value)
				case "RecordID_2":
					require.Equal(istructs.RecordID(2), value)
				default:
					t.Fail()
				}
				cnt++
			})

		require.Equal(2, cnt)
	})

	t.Run("RecordIDs must iterate not null IDs", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutRecordID("RecordID", 1)
		row.PutRecordID("RecordID_2", istructs.NullRecordID)

		require.NoError(row.build())

		cnt := 0
		row.RecordIDs(false,
			func(name string, value istructs.RecordID) {
				switch name {
				case "RecordID":
					require.Equal(istructs.RecordID(1), value)
				default:
					t.Fail()
				}
				cnt++
			})

		require.Equal(1, cnt)
	})
}

func Test_rowType_maskValues(t *testing.T) {
	require := require.New(t)

	t.Run("maskValues must hide all rows data", func(t *testing.T) {
		row := newTestRow()

		row.maskValues()

		require.Equal(int32(0), row.AsInt32("int32"))
		require.Equal(int64(0), row.AsInt64("int64"))
		require.Equal(float32(0), row.AsFloat32("float32"))
		require.Equal(float64(0), row.AsFloat64("float64"))
		require.Nil(row.AsBytes("bytes"))
		require.Equal("*", row.AsString("string"))
		require.Equal(appdef.NullQName, row.AsQName("QName"))
		require.Equal(false, row.AsBool("bool"))
		require.Equal(istructs.NullRecordID, row.AsRecordID("RecordID"))
	})
}

func Test_rowType_FieldNames(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("new [or null] row must have hot fields", func(t *testing.T) {
		row := newRow(test.AppCfg)

		cnt := 0
		row.FieldNames(func(fieldName string) {
			cnt++
		})
		require.Equal(0, cnt)
	})

	t.Run("new test row must have only QName field", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		cnt := 0
		row.FieldNames(func(fieldName string) {
			require.Equal(appdef.SystemField_QName, fieldName)
			cnt++
		})
		require.Equal(1, cnt)
	})

	t.Run("filled test row must iterate all fields without duplicates", func(t *testing.T) {
		row := newTestRow()

		cnt := 0
		names := make(map[string]bool)
		row.FieldNames(func(fieldName string) {
			require.False(names[fieldName])
			names[fieldName] = true
			cnt++
		})
		require.Equal(10, cnt) // QName + nine user fields for simple types
	})
}

func Test_rowType_BuildErrors(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("Put××× unknown field name must have build error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutInt32("unknown", 1)
		require.ErrorIs(row.build(), ErrNameNotFound)
	})

	t.Run("Put××× invalid field value type must have build error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutString("int32", "a")
		require.ErrorIs(row.build(), ErrWrongFieldType)
	})

	t.Run("PutString to []byte type must collect convert error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutString("bytes", "some string")

		require.ErrorIs(row.build(), ErrWrongFieldType)
	})

	t.Run("PutQName invalid QName must have build error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutString("QName", "zZz")

		require.ErrorIs(row.build(), ErrWrongFieldType)
	})
}

func Test_rowType_Nils(t *testing.T) {
	require := require.New(t)
	test := test()

	t.Run("must be empty nils if no nil assignment", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutInt32("int32", 8)
		require.NoError(row.build())
		require.Empty(row.nils)
	})

	t.Run("check nils", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		t.Run("check first nil", func(t *testing.T) {
			row.PutInt32("int32", 8)
			row.PutChars("bytes", "")
			require.NoError(row.build())
			require.Len(row.nils, 1)
			require.Contains(row.nils, "bytes")
		})

		t.Run("check second nil", func(t *testing.T) {
			row.PutChars("string", "")
			require.NoError(row.build())
			require.Len(row.nils, 2)
			require.Contains(row.nils, "bytes")
			require.Contains(row.nils, "string")
		})

		t.Run("check repeat nil", func(t *testing.T) {
			row.PutChars("bytes", "")
			require.NoError(row.build())
			require.Len(row.nils, 2)
			require.Contains(row.nils, "bytes")
			require.Contains(row.nils, "string")
		})

		t.Run("check nils are kept", func(t *testing.T) {
			row.PutInt32("int32", 888)
			require.NoError(row.build())
			require.Len(row.nils, 2)
			require.Contains(row.nils, "bytes")
			require.Contains(row.nils, "string")
		})

		t.Run("check nil can be reassigned", func(t *testing.T) {
			row.PutBytes("bytes", []byte{0})
			require.NoError(row.build())
			require.Len(row.nils, 1)
			require.Contains(row.nils, "string")
		})
	})

	t.Run("check nil assignment", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)
		row.PutInt32("int32", 0)
		row.PutInt64("int64", 0)
		row.PutFloat32("float32", 0)
		row.PutFloat64("float64", 0)
		row.PutBytes("bytes", []byte{})
		row.PutString("string", "")
		row.PutQName("QName", appdef.NullQName)
		row.PutBool("bool", false)
		row.PutRecordID("RecordID", istructs.NullRecordID)

		require.NoError(row.build())

		require.True(row.HasValue("int32"))
		require.True(row.HasValue("int64"))
		require.True(row.HasValue("float32"))
		require.True(row.HasValue("float64"))
		require.False(row.HasValue("bytes"))
		require.False(row.HasValue("string"))
		require.True(row.HasValue("QName"))
		require.True(row.HasValue("bool"))
		require.True(row.HasValue("RecordID"))

		cnt := 0
		row.dyB.IterateFields(nil, func(name string, newData interface{}) bool {
			switch name {
			case "int32", "int64", "float32", "float64":
				require.Zero(newData)
			case "QName":
				var nullQNameBytes = []uint8([]byte{0x0, 0x0})
				require.Equal(nullQNameBytes, newData)
			case "bool":
				require.False(newData.(bool))
			case "RecordID":
				require.EqualValues(istructs.NullRecordID, newData)
			default:
				require.Fail("unexpected field", "field name: «%s»", name)
			}
			cnt++
			return true
		})

		require.Equal(7, cnt)

		require.Len(row.nils, 2)
		require.Contains(row.nils, "bytes")
		require.Contains(row.nils, "string")
	})
}
