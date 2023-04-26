/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_RecordsRead(t *testing.T) {
	require := require.New(t)
	test := test()

	storage := teststore.NewStorage()
	storageProvider := teststore.NewStorageProvider(storage)

	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

	app, err := provider.AppStructs(test.appName)
	require.NoError(err)

	const (
		minTestRecordID  istructs.RecordID = 100500
		testRecordsCount                   = 10000
		maxTestRecordID                    = minTestRecordID + testRecordsCount
	)

	t.Run("prepare records to read", func(t *testing.T) {
		batch := make([]recordBatchItemType, 0)
		for id := minTestRecordID; id <= maxTestRecordID; id++ {
			rec := newTestCRecord(id)
			data, err := rec.storeToBytes()
			require.NoError(err)
			batch = append(batch, recordBatchItemType{id, data})
		}
		err := app.Records().(*appRecordsType).putRecordsBatch(test.workspace, batch)
		require.NoError(err)
	})

	t.Run("test once read records", func(t *testing.T) {
		mustExists := func(id istructs.RecordID) {
			t.Run(fmt.Sprintf("must ok read exists record %v", id), func(t *testing.T) {
				rec, err := app.Records().Get(test.workspace, true, id)
				require.NoError(err)
				testTestCRec(t, rec, id)
			})
		}

		mustExists(minTestRecordID)
		mustExists((minTestRecordID + maxTestRecordID) / 2)
		mustExists(maxTestRecordID)

		mustAbsent := func(id istructs.RecordID) {
			t.Run(fmt.Sprintf("must ok read not exists record %v", id), func(t *testing.T) {
				rec, err := app.Records().Get(test.workspace, true, id)
				require.NoError(err)
				require.Equal(schemas.NullQName, rec.QName())
				require.Equal(id, rec.ID())
			})
		}

		mustAbsent(istructs.NullRecordID)
		mustAbsent(minTestRecordID - 1)
		mustAbsent(maxTestRecordID + 1)
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
						testTestCRec(t, rec.Record, rec.ID)
					} else {
						require.Equal(schemas.NullQName, rec.Record.QName())
					}
				}
			}
		})

		t.Run("test batch read records from random intervals", func(t *testing.T) {
			const maxIntervalLength = 16
			rand.Seed(time.Now().UnixNano())
			recs := make([]istructs.RecordGetBatchItem, maxGetBatchRecordCount)
			for i := 0; i < maxGetBatchRecordCount; {
				l := rand.Intn(maxIntervalLength) + 1
				if i+l > maxGetBatchRecordCount {
					l = maxGetBatchRecordCount - i
				}
				id := minTestRecordID + istructs.RecordID(rand.Intn(testRecordsCount-l))
				for j := 0; j < l; j++ {
					recs[i].ID = id + istructs.RecordID(j)
					i++
				}
			}

			err := app.Records().GetBatch(test.workspace, true, recs)
			require.NoError(err)

			for _, rec := range recs {
				require.Equal(rec.ID, rec.Record.ID())
				testTestCRec(t, rec.Record, rec.ID)
			}
		})
	})

	t.Run("must fail if too large batch read records", func(t *testing.T) {
		recs := make([]istructs.RecordGetBatchItem, maxGetBatchRecordCount+1)
		for id := minTestRecordID; id < minTestRecordID+maxGetBatchRecordCount+1; id++ {
			recs[id-minTestRecordID].ID = id
		}
		err := app.Records().GetBatch(test.workspace, true, recs)
		require.ErrorIs(err, ErrMaxGetBatchRecordCountExceeds)
	})

	t.Run("must fail batch read records if storage batch failed", func(t *testing.T) {
		testError := fmt.Errorf("test error")
		testID := istructs.RecordID(100500)
		_, cc := splitRecordID(testID)

		storage.ScheduleGetError(testError, nil, cc)
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		app, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)

		recs := make([]istructs.RecordGetBatchItem, 3)
		recs[0].ID = testID - 1
		recs[1].ID = testID
		recs[2].ID = testID + 1

		err = app.Records().GetBatch(test.workspace, true, recs)
		require.ErrorIs(err, testError)
	})

	t.Run("must fail batch read records if storage returns damaged data", func(t *testing.T) {
		testID := istructs.RecordID(100500)
		_, cc := splitRecordID(testID)

		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, nil, cc)
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		app, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)

		rec := newTestCRecord(testID)
		data, err := rec.storeToBytes()
		require.NoError(err)
		app.Records().(*appRecordsType).putRecord(test.workspace, testID, data)

		recs := make([]istructs.RecordGetBatchItem, 3)
		recs[0].ID = testID - 1
		recs[1].ID = testID
		recs[2].ID = testID + 1

		err = app.Records().GetBatch(test.workspace, true, recs)
		require.ErrorIs(err, ErrUnknownCodec)
	})
}
