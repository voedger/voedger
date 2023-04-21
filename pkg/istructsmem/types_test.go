/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
)

func Test_dynoBufValue(t *testing.T) {
	require := require.New(t)
	test := test()

	row := newTestRow()

	t.Run("test int32", func(t *testing.T) {
		v, err := row.dynoBufValue(int32(7), istructs.DataKind_int32)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue(float64(7), istructs.DataKind_int32)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue("7", istructs.DataKind_int32)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test int64", func(t *testing.T) {
		v, err := row.dynoBufValue(int64(7), istructs.DataKind_int64)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue(float64(7), istructs.DataKind_int64)
		require.NoError(err)
		require.EqualValues(7, v)

		v, err = row.dynoBufValue("7", istructs.DataKind_int64)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test float32", func(t *testing.T) {
		v, err := row.dynoBufValue(float32(7.7), istructs.DataKind_float32)
		require.NoError(err)
		require.EqualValues(7.7, v)

		v, err = row.dynoBufValue(float64(7.7), istructs.DataKind_float32)
		require.NoError(err)
		require.EqualValues(7.7, v)

		v, err = row.dynoBufValue("7.7", istructs.DataKind_float32)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test float64", func(t *testing.T) {
		v, err := row.dynoBufValue(float64(7.7), istructs.DataKind_float64)
		require.NoError(err)
		require.EqualValues(7.7, v)

		v, err = row.dynoBufValue(7, istructs.DataKind_float64)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test []byte", func(t *testing.T) {
		v, err := row.dynoBufValue([]byte{1, 2, 3}, istructs.DataKind_bytes)
		require.NoError(err)
		require.Equal([]byte{1, 2, 3}, v)

		v, err = row.dynoBufValue("AQIDBA==", istructs.DataKind_bytes)
		require.NoError(err)
		require.Equal([]byte{1, 2, 3, 4}, v)

		v, err = row.dynoBufValue("ups", istructs.DataKind_bytes)
		require.Error(err) // base64 convert error
		require.Nil(v)

		v, err = row.dynoBufValue(7, istructs.DataKind_bytes)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test QName", func(t *testing.T) {
		id, _ := test.AppCfg.qNames.GetID(test.saleCmdName)
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(id))

		v, err := row.dynoBufValue(test.saleCmdName, istructs.DataKind_QName)
		require.NoError(err)
		require.EqualValues(b, v)

		v, err = row.dynoBufValue(test.saleCmdName.String(), istructs.DataKind_QName)
		require.NoError(err)
		require.EqualValues(b, v)

		v, err = row.dynoBufValue(istructs.NewQName("test", "unknown"), istructs.DataKind_QName)
		require.ErrorIs(err, qnames.ErrNameNotFound)
		require.Nil(v)

		v, err = row.dynoBufValue("test.unknown", istructs.DataKind_QName)
		require.ErrorIs(err, qnames.ErrNameNotFound)
		require.Nil(v)

		v, err = row.dynoBufValue("ups!", istructs.DataKind_QName)
		require.Error(err) // invalid QName
		require.Nil(v)

		v, err = row.dynoBufValue(7, istructs.DataKind_bytes)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test bool", func(t *testing.T) {
		v, err := row.dynoBufValue(false, istructs.DataKind_bool)
		require.NoError(err)
		require.Equal(false, v)

		v, err = row.dynoBufValue(true, istructs.DataKind_bool)
		require.NoError(err)
		require.Equal(true, v)

		v, err = row.dynoBufValue(7, istructs.DataKind_bool)
		require.ErrorIs(err, ErrWrongFieldType)
		require.Nil(v)
	})

	t.Run("test int64", func(t *testing.T) {
		v, err := row.dynoBufValue(istructs.NullRecordID, istructs.DataKind_RecordID)
		require.NoError(err)
		require.EqualValues(istructs.NullRecordID, v)

		v, err = row.dynoBufValue(istructs.RecordID(100500700), istructs.DataKind_RecordID)
		require.NoError(err)
		require.EqualValues(100500700, v)

		v, err = row.dynoBufValue(float64(100500700), istructs.DataKind_RecordID)
		require.NoError(err)
		require.EqualValues(100500700, v)

		v, err = row.dynoBufValue("100500700", istructs.DataKind_RecordID)
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

		v, err := row.dynoBufValue(testRec, istructs.DataKind_Record)
		require.NoError(err)
		checkRecord(v)

		v, err = row.dynoBufValue("ups", istructs.DataKind_Record)
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

		v, err := row.dynoBufValue(testEvent, istructs.DataKind_Event)
		require.NoError(err)
		checkEvent(v)

		v, err = row.dynoBufValue("ups", istructs.DataKind_Event)
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
		require.Equal(istructs.NullQName, row.AsQName("QName"))
		require.Equal(false, row.AsBool("bool"))
		require.Equal(istructs.NullRecordID, row.AsRecordID("RecordID"))

		val := newEmptyViewValue()
		require.Equal(istructs.IDbEvent(nil), val.AsEvent(test.testViewRecord.valueFields.event))
		rec := val.AsRecord(test.testViewRecord.valueFields.record)
		require.Equal(istructs.NullQName, rec.QName())
	})

	t.Run("PutNumber to numeric-type fields must be available (json)", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutNumber("int32", 1)
		row.PutNumber("int64", 2)
		row.PutNumber("float32", 3)
		row.PutNumber("float64", 4)
		row.PutNumber("RecordID", 5)

		_, err := row.build()
		require.NoError(err)

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
		row.PutChars("bytes", "AQIDBA==")

		_, err := row.build()
		require.NoError(err)

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
		_, err := row.build()
		require.NoError(err)

		rec := row.AsRecord(test.testViewRecord.valueFields.record)
		require.NotNil(rec)
		require.Equal(istructs.NullQName, rec.QName())
		require.Equal(istructs.NullRecordID, rec.AsRecordID(istructs.SystemField_ID))
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
			_, err := row.build()
			require.ErrorIs(err, ErrNameNotFound)
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
			_, err := row.build()
			require.ErrorIs(err, ErrWrongFieldType)
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

		_, err := row.build()
		require.ErrorIs(err, ErrWrongFieldType)
	})

	t.Run("PutQName with unknown QName value must be error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutQName("QName", istructs.NewQName("unknown", "unknown"))

		_, err := row.build()
		require.ErrorIs(err, qnames.ErrNameNotFound)
	})

	t.Run("PutChars error handling", func(t *testing.T) {
		t.Run("PutChars to non-char type fields must be error", func(t *testing.T) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)

			row.PutChars("int32", "29")

			_, err := row.build()
			require.ErrorIs(err, ErrWrongFieldType)
		})

		t.Run("PutChars to QName-type fields non-covertable value must be error", func(t *testing.T) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)

			row.PutChars("QName", "wellcome.2.error")

			_, err := row.build()
			require.ErrorIs(err, istructs.ErrInvalidQNameStringRepresentation)
		})

		t.Run("PutChars to bytes-type fields non-covertable base64 value must be error", func(t *testing.T) {
			row := newRow(test.AppCfg)
			row.setQName(test.testRow)

			row.PutChars("bytes", "wellcome.2.error")

			_, err := row.build()
			require.Error(err)
		})
	})

	t.Run("Multiply Put××× errors must be concatenated in build error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutFloat32("unknown_field", 555.5)
		row.PutInt32("int64", 1)

		_, err := row.build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorIs(err, ErrWrongFieldType)
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

		_, err := row.build()
		require.NoError(err)

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

		_, err := row.build()
		require.NoError(err)

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
		require.Equal(istructs.NullQName, row.AsQName("QName"))
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
			require.Equal(istructs.SystemField_QName, fieldName)
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
		_, err := row.build()
		require.Error(err)
	})

	t.Run("Put××× invalid field value type must have build error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutString("int32", "a")
		_, err := row.build()
		require.Error(err)
	})

	t.Run("PutString to []byte type must collect convert error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutString("bytes", "some string")

		_, err := row.build()
		require.Error(err)
	})

	t.Run("PutQName invalid QName must have build error", func(t *testing.T) {
		row := newRow(test.AppCfg)
		row.setQName(test.testRow)

		row.PutString("QName", "zZz")
		_, err := row.build()
		require.Error(err)
	})
}
