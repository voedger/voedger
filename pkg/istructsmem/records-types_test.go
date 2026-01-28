/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/isequencer"

	gojson "encoding/json"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
)

func Test_RecordsRead(t *testing.T) {
	require := require.New(t)
	test := newTest()

	app := test.AppStructs

	const (
		minTestRecordID  istructs.RecordID = 100500
		testRecordsCount                   = 10000
		maxTestRecordID                    = minTestRecordID + testRecordsCount
	)

	t.Run("prepare records to read", func(t *testing.T) {
		batch := make([]recordBatchItemType, 0)
		for id := minTestRecordID; id <= maxTestRecordID; id++ {
			rec := test.newTestCRecord(id)
			data := rec.storeToBytes()
			batch = append(batch, recordBatchItemType{id, data, true})
		}
		err := app.Records().(*appRecordsType).putRecordsBatch(test.workspace, batch, false)
		require.NoError(err)
	})

	t.Run("test once read records", func(t *testing.T) {
		mustExists := func(id istructs.RecordID) {
			t.Run(fmt.Sprintf("must ok read exists record %v", id), func(t *testing.T) {
				rec, err := app.Records().Get(test.workspace, true, id)
				require.NoError(err)
				test.testTestCRec(t, rec, id)
			})
		}

		mustExists(minTestRecordID)
		mustExists((minTestRecordID + maxTestRecordID) / 2)
		mustExists(maxTestRecordID)

		mustAbsent := func(id istructs.RecordID) {
			t.Run(fmt.Sprintf("must ok read not exists record %v", id), func(t *testing.T) {
				rec, err := app.Records().Get(test.workspace, true, id)
				require.NoError(err)
				require.Equal(appdef.NullQName, rec.QName())
				require.Equal(id, rec.ID())
			})
		}

		mustAbsent(istructs.NullRecordID)
		mustAbsent(minTestRecordID - 1)
		mustAbsent(maxTestRecordID + 1)

		t.Run("should be error", func(t *testing.T) {
			t.Run("if storage get failed", func(t *testing.T) {
				id := maxTestRecordID + 1
				_, cc := recordKey(0, id)
				testError := errors.New("test storage Get failed")
				test.Storage.ScheduleGetError(testError, nil, cc)
				defer test.Storage.Reset()

				rec, err := app.Records().Get(test.workspace, true, id)
				require.Error(err, require.Is(testError), require.HasAll(test.workspace, id))
				require.Equal(appdef.NullQName, rec.QName())
				require.Equal(id, rec.ID())
			})

			t.Run("if storage returns damaged data", func(t *testing.T) {
				const badCodec byte = 255
				_, cc := recordKey(0, minTestRecordID)

				test.Storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = badCodec /* error here */ }, nil, cc)
				defer test.Storage.Reset()

				rec, err := app.Records().Get(test.workspace, true, minTestRecordID)
				require.Error(err, require.Is(ErrUnknownCodecError), require.HasAll(badCodec, test.workspace, minTestRecordID))
				require.Equal(appdef.NullQName, rec.QName())
				require.Equal(minTestRecordID, rec.ID())
			})
		})
	})

	t.Run("test batch read records", func(t *testing.T) {

		t.Run("test sequence batch read records", func(t *testing.T) {
			for minID := minTestRecordID - 500; minID < maxTestRecordID+500; minID += maxGetBatchRecordCount {
				recs := make([]istructs.RecordGetBatchItem, maxGetBatchRecordCount)
				for id := minID; id < minID+maxGetBatchRecordCount; id++ {
					recs[id-minID].ID = id
				}
				err := app.Records().GetBatch(test.workspace, true, recs)
				require.NoError(err)

				for i, rec := range recs {
					require.Equal(minID+istructs.RecordID(i), rec.ID)
					require.Equal(rec.ID, rec.Record.ID())
					if (rec.ID >= minTestRecordID) && (rec.ID <= maxTestRecordID) {
						test.testTestCRec(t, rec.Record, rec.ID)
					} else {
						require.Equal(appdef.NullQName, rec.Record.QName())
					}
				}
			}
		})

		// nolint: staticcheck
		t.Run("test batch read records from random intervals", func(t *testing.T) {
			const maxIntervalLength = 16
			recs := make([]istructs.RecordGetBatchItem, maxGetBatchRecordCount)
			for i := 0; i < maxGetBatchRecordCount; {
				l := rand.Intn(maxIntervalLength) + 1
				if i+l > maxGetBatchRecordCount {
					l = maxGetBatchRecordCount - i
				}
				id := minTestRecordID + istructs.RecordID(rand.Intn(testRecordsCount-l))
				for j := range l {
					recs[i].ID = id + istructs.RecordID(j)
					i++
				}
			}

			err := app.Records().GetBatch(test.workspace, true, recs)
			require.NoError(err)

			for _, rec := range recs {
				require.Equal(rec.ID, rec.Record.ID())
				test.testTestCRec(t, rec.Record, rec.ID)
			}
		})
	})

	t.Run("must fail if too large batch read records", func(t *testing.T) {
		const tooBig = maxGetBatchRecordCount + 1
		recs := make([]istructs.RecordGetBatchItem, tooBig)
		for id := minTestRecordID; id < minTestRecordID+tooBig; id++ {
			recs[id-minTestRecordID].ID = id
		}
		err := app.Records().GetBatch(test.workspace, true, recs)
		require.Error(err, require.Is(ErrMaxGetBatchSizeExceedsError), require.Has(tooBig))
	})

	t.Run("must fail batch read records if storage batch failed", func(t *testing.T) {
		testError := errors.New("test error")
		testID := istructs.RecordID(100500)
		_, cc := recordKey(0, testID)

		test.Storage.ScheduleGetError(testError, nil, cc)
		defer test.Storage.Reset()

		recs := make([]istructs.RecordGetBatchItem, 3)
		recs[0].ID = testID - 1
		recs[1].ID = testID
		recs[2].ID = testID + 1

		err := app.Records().GetBatch(test.workspace, true, recs)
		require.ErrorIs(err, testError)
	})

	t.Run("must fail batch read records if storage returns damaged data", func(t *testing.T) {
		testID := istructs.RecordID(100500)
		_, cc := recordKey(0, testID)

		const badCodec byte = 255

		test.Storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = badCodec /* error here */ }, nil, cc)
		defer test.Storage.Reset()

		rec := test.newTestCRecord(testID)
		data := rec.storeToBytes()
		err := app.Records().(*appRecordsType).putRecord(test.workspace, testID, data)
		require.NoError(err)

		recs := make([]istructs.RecordGetBatchItem, 3)
		recs[0].ID = testID - 1
		recs[1].ID = testID
		recs[2].ID = testID + 1

		err = app.Records().GetBatch(test.workspace, true, recs)
		require.Error(err, require.Is(ErrUnknownCodecError), require.Has(badCodec))
	})
}

func Test_RecordsPutJSON(t *testing.T) {
	require := require.New(t)
	test := newTest()

	storage := teststore.NewStorage(test.appName)
	storageProvider := teststore.NewStorageProvider(storage)

	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)

	app, err := provider.BuiltIn(test.appName)
	require.NoError(err)

	json := make(map[appdef.FieldName]any)
	json[appdef.SystemField_QName] = test.testCDoc.String()
	json[appdef.SystemField_ID] = istructs.RecordID(100500)
	json["int32"] = int32(1)
	json["int64"] = int64(2)
	json["float32"] = float32(3)
	json["float64"] = float64(4)
	// cspell:disable
	json["bytes"] = `AQIDBA==`
	// cspell:enable
	json["string"] = `naked 🔫`
	json["QName"] = test.testCRec.String()
	json["bool"] = true
	json["RecordID"] = istructs.RecordID(100501)

	t.Run("should be ok to put record from JSON", func(t *testing.T) {
		err := app.Records().PutJSON(test.workspace, json)
		require.NoError(err)

		t.Run("should be ok to read record", func(t *testing.T) {
			r, err := app.Records().Get(test.workspace, true, 100500)
			require.NoError(err)

			require.EqualValues(test.testCDoc, r.QName())
			require.EqualValues(100500, r.ID())
			require.EqualValues(1, r.AsInt32("int32"))
			require.EqualValues(2, r.AsInt64("int64"))
			require.EqualValues(3, r.AsFloat32("float32"))
			require.EqualValues(4, r.AsFloat64("float64"))
			require.Equal([]byte{1, 2, 3, 4}, r.AsBytes("bytes"))
			require.Equal(`naked 🔫`, r.AsString("string"))
			require.Equal(test.testCRec, r.AsQName("QName"))
			require.True(r.AsBool("bool"))
			require.EqualValues(100501, r.AsRecordID("RecordID"))
		})
	})

	t.Run("numbers as json.Number", func(t *testing.T) {
		json := make(map[appdef.FieldName]any)
		json[appdef.SystemField_QName] = test.testCDoc.String()
		json[appdef.SystemField_ID] = gojson.Number("100500")
		json["int32"] = gojson.Number("1")
		json["int64"] = gojson.Number("2")
		json["float32"] = gojson.Number("3")
		json["float64"] = gojson.Number("4")
		json["RecordID"] = gojson.Number("100501")

		err := app.Records().PutJSON(test.workspace, json)
		require.NoError(err)

		r, err := app.Records().Get(test.workspace, true, 100500)
		require.NoError(err)

		require.EqualValues(test.testCDoc, r.QName())
		require.EqualValues(100500, r.ID())
		require.EqualValues(1, r.AsInt32("int32"))
		require.EqualValues(2, r.AsInt64("int64"))
		require.EqualValues(3, r.AsFloat32("float32"))
		require.EqualValues(4, r.AsFloat64("float64"))
		require.EqualValues(100501, r.AsRecordID("RecordID"))
	})

	t.Run("should fail to put record with invalid QName", func(t *testing.T) {
		json := make(map[appdef.FieldName]any)

		json[appdef.SystemField_QName] = appdef.NullQName.String()
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrFieldIsEmptyError),
			require.Has(appdef.SystemField_QName))

		json[appdef.SystemField_QName] = 123
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrWrongFieldTypeError), require.Has(appdef.SystemField_QName))

		json[appdef.SystemField_QName] = `naked 🔫`
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(appdef.ErrConvertError), require.Has(appdef.SystemField_QName))

		json[appdef.SystemField_QName] = test.testObj.String()
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err,
			require.Is(ErrWrongTypeError), require.Has(test.testObj))
	})

	t.Run("should fail to put record with invalid RecordID", func(t *testing.T) {
		json := make(map[appdef.FieldName]any)
		json[appdef.SystemField_QName] = test.testCDoc.String()

		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrFieldIsEmptyError),
			require.HasAll(test.testCDoc, appdef.SystemField_ID))

		json[appdef.SystemField_ID] = int64(0)
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrFieldIsEmptyError),
			require.HasAll(test.testCDoc, appdef.SystemField_ID))
		require.ErrorContains(err, appdef.SystemField_ID)

		json[appdef.SystemField_ID] = gojson.Number("0")
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrFieldIsEmptyError),
			require.HasAll(test.testCDoc, appdef.SystemField_ID))

		json[appdef.SystemField_ID] = int64(1)
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrUnexpectedRawRecordIDError),
			require.HasAll(test.testCDoc, appdef.SystemField_ID, 1))

		json[appdef.SystemField_ID] = gojson.Number("1")
		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrUnexpectedRawRecordIDError),
			require.HasAll(test.testCDoc, appdef.SystemField_ID, 1))
	})

	t.Run("should fail to put record with invalid data", func(t *testing.T) {
		json := make(map[appdef.FieldName]any)
		json[appdef.SystemField_QName] = test.testCDoc.String()
		json[appdef.SystemField_ID] = float64(100500)

		json["unknown"] = `naked 🔫`

		err = app.Records().PutJSON(test.workspace, json)
		require.Error(err, require.Is(ErrNameNotFoundError), require.Has("unknown"))
	})
}
