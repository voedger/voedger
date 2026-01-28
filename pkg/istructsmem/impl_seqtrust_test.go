/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package istructsmem

import (
	"context"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestSequencesTrustLevel(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1
	qNameCDoc := appdef.NewQName("test", "cdoc")

	// create app configuration
	appConfigs := func() AppConfigsType {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))

		saleParamsName := appdef.NewQName("test", "SaleParams")

		wsb.AddCDoc(qNameCDoc)

		saleParamsDoc := wsb.AddODoc(saleParamsName)
		saleParamsDoc.
			AddField("Buyer", appdef.DataKind_string, true).
			AddField("Age", appdef.DataKind_int8, false).
			AddField("Height", appdef.DataKind_float32, false).
			AddField("isHuman", appdef.DataKind_bool, false).
			AddField("Photo", appdef.DataKind_bytes, false)

		goodRec := wsb.AddORecord(appdef.NewQName("test", "Good"))
		goodRec.
			AddField("Name", appdef.DataKind_string, true).
			AddField("Code", appdef.DataKind_int64, true).
			AddField("Weight", appdef.DataKind_float64, false)

		qNameCmdTestSale := appdef.NewQName("test", "Sale")
		wsb.AddCommand(qNameCmdTestSale).
			SetParam(saleParamsName)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		cfg.Resources.Add(NewCommandFunction(qNameCmdTestSale, NullCommandExec))

		return cfgs
	}()

	provider := Provide(appConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	bld := app.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				HandlingPartition: 55,
				PLogOffset:        10000,
				Workspace:         1234,
				WLogOffset:        1000,
				QName:             appdef.NewQName("test", "Sale"),
				RegisteredAt:      100500,
			},
			Device:   762,
			SyncedAt: 1005001,
		})

	cmd := bld.ArgumentObjectBuilder()

	cmd.PutRecordID(appdef.SystemField_ID, 1)
	cmd.PutString("Buyer", "Carlson 哇\"呀呀") // to test unicode issues
	cmd.PutInt8("Age", 33)
	cmd.PutFloat32("Height", 1.75)
	cmd.PutBytes("Photo", []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 9, 8, 7, 6, 4, 4, 3, 2, 1, 0})

	bld.CUDBuilder().Create(qNameCDoc).PutRecordID(appdef.SystemField_ID, 2)

	rawEvent, buildErr := bld.BuildRawEvent()
	require.NoError(buildErr)

	// PutPLog
	// 1st PutPLog on seqTrustLevel_0 -> ok
	pLogEvent, err := app.Events().PutPlog(rawEvent, buildErr, NewIDGenerator())
	require.NoError(err)

	t.Run("plog", func(t *testing.T) {
		t.Run("trust level 0", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_0
			t.Run("panic on write the same PLogOffset", func(t *testing.T) {
				ev, err := app.Events().PutPlog(rawEvent, buildErr, NewIDGenerator())
				require.ErrorIs(err, ErrSequencesViolation)
				require.Nil(ev)
			})
		})
		t.Run("trust level 1", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_1
			t.Run("panic on write the same PLogOffset", func(t *testing.T) {
				ev, err := app.Events().PutPlog(rawEvent, buildErr, NewIDGenerator())
				require.ErrorIs(err, ErrSequencesViolation)
				require.Nil(ev)
			})
		})

		t.Run("trust level 2", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_2
			t.Run("ok to overwrite PLog (dangerous)", func(t *testing.T) {
				_, err := app.Events().PutPlog(rawEvent, buildErr, NewIDGenerator())
				require.NoError(err)
			})
		})
	})

	err = app.Records().Apply(pLogEvent)
	require.NoError(err)

	t.Run("records", func(t *testing.T) {
		t.Run("trust level 0", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_0
			t.Run("panic on write the same RecordIDs", func(t *testing.T) {
				err := app.Records().Apply(pLogEvent)
				require.ErrorIs(err, ErrSequencesViolation)
			})
		})
		t.Run("trust level 1", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_1
			t.Run("ok to overwrite records (dangerous)", func(t *testing.T) {
				require.NoError(app.Records().Apply(pLogEvent))
			})
		})

		t.Run("trust level 2", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_2
			t.Run("ok to overwrite records (dangerous)", func(t *testing.T) {
				require.NoError(app.Records().Apply(pLogEvent))
			})
		})
	})

	// PutWLog
	// 1st PutPLog on seqTrustLevel_0 -> ok
	err = app.Events().PutWlog(pLogEvent)
	require.NoError(err)

	t.Run("wlog", func(t *testing.T) {
		t.Run("trust level 0", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_0
			t.Run("panic on overwrite the same WLogOffset", func(t *testing.T) {
				err := app.Events().PutWlog(pLogEvent)
				require.ErrorIs(err, ErrSequencesViolation)
			})
		})
		t.Run("trust level 1", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_1
			t.Run("panic on write the same WLogOffset", func(t *testing.T) {
				err := app.Events().PutWlog(pLogEvent)
				require.ErrorIs(err, ErrSequencesViolation)
			})
		})

		t.Run("trust level 2", func(t *testing.T) {
			app.(*appStructsType).seqTrustLevel = isequencer.SequencesTrustLevel_2
			t.Run("ok to overwrite WLog (dangerous)", func(t *testing.T) {
				err := app.Events().PutWlog(pLogEvent)
				require.NoError(err)
			})
		})
	})
}

func TestEventReapplier(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	// create app configuration
	appConfigs := func() AppConfigsType {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		saleParamsName := appdef.NewQName("test", "SaleParams")
		wsb.AddODoc(saleParamsName)
		qNameCmdTestSale := appdef.NewQName("test", "Sale")
		wsb.AddCommand(qNameCmdTestSale).
			SetParam(saleParamsName)
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		cfg.Resources.Add(NewCommandFunction(qNameCmdTestSale, NullCommandExec))

		return cfgs
	}()

	storageProvider := simpleStorageProvider()
	provider := Provide(appConfigs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	bld := app.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				HandlingPartition: 55,
				PLogOffset:        10000,
				Workspace:         1234,
				WLogOffset:        1000,
				QName:             appdef.NewQName("test", "Sale"),
				RegisteredAt:      100500,
			},
			Device:   762,
			SyncedAt: 1005001,
		})
	cmd := bld.ArgumentObjectBuilder()
	cmd.PutRecordID(appdef.SystemField_ID, 1)
	rawEvent, buildErr := bld.BuildRawEvent()
	require.NoError(buildErr)

	pLogEvent, err := app.Events().PutPlog(rawEvent, buildErr, NewIDGenerator())
	require.NoError(err)

	err = app.Records().Apply(pLogEvent)
	require.NoError(err)

	err = app.Events().PutWlog(pLogEvent)
	require.NoError(err)

	t.Run("ok to re-apply the event loaded from the db", func(t *testing.T) {
		t.Run("plog cache", func(t *testing.T) {
			var dbPLogEvent istructs.IPLogEvent
			err = app.Events().ReadPLog(context.Background(), 55, 10000, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
				require.Nil(dbPLogEvent)
				dbPLogEvent = event
				return nil
			})
			require.NoError(err)
			reapplier := app.GetEventReapplier(dbPLogEvent)
			require.NoError(reapplier.ApplyRecords())
			require.NoError(reapplier.PutWLog())
		})
		t.Run("initially read from storage", func(t *testing.T) {
			provider := Provide(appConfigs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
			app, err := provider.BuiltIn(appName)
			require.NoError(err)
			var dbPLogEvent istructs.IPLogEvent
			err = app.Events().ReadPLog(context.Background(), 55, 10000, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
				require.Nil(dbPLogEvent)
				dbPLogEvent = event
				return nil
			})
			require.NoError(err)
			reapplier := app.GetEventReapplier(dbPLogEvent)
			require.NoError(reapplier.ApplyRecords())
			require.NoError(reapplier.PutWLog())
		})
	})
}
