/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/voedger/voedger/pkg/appdef/builder"
	log "github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/singletons"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

func TestEventBuilder(t *testing.T) {
	testEventBuilderCore(t, true)
}

func TestEventBuilderNoCache(t *testing.T) {
	testEventBuilderCore(t, false)
}

func testEventBuilderCore(t *testing.T, cachedPLog bool) {
	require := require.New(t)
	test := test()

	if !cachedPLog {
		// switch off plog cache
		test.AppCfg.Params.PLogEventCacheSize = 0
	}

	app := test.AppStructs

	var rawEvent istructs.IRawEvent
	var buildErr error

	var saleID istructs.RecordID
	var basketID istructs.RecordID
	var goodsID [2]istructs.RecordID
	var photoID istructs.RecordID
	var remarkID istructs.RecordID

	t.Run("I. Build raw event demo", func(t *testing.T) {
		// gets event builder
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.saleCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})

		t.Run("make command object", func(t *testing.T) {
			cmd := bld.ArgumentObjectBuilder()

			cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
			cmd.PutString(test.buyerIdent, test.buyerValue)
			cmd.PutInt32(test.ageIdent, test.ageValue)
			cmd.PutFloat32(test.heightIdent, test.heightValue)
			cmd.PutBool(test.humanIdent, test.humanValue)
			cmd.PutBytes(test.photoIdent, test.photoValue)

			basket := cmd.ChildBuilder(test.basketIdent)
			basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)
			for i := 0; i < test.goodCount; i++ {
				good := basket.ChildBuilder(test.goodIdent)
				good.PutRecordID(appdef.SystemField_ID, test.tempGoodsID[i])
				good.PutRecordID(test.saleIdent, test.tempSaleID)
				good.PutString(test.nameIdent, test.goodNames[i])
				good.PutInt64(test.codeIdent, test.goodCodes[i])
				good.PutFloat64(test.weightIdent, test.goodWeights[i])
			}

			cmdSec := bld.ArgumentUnloggedObjectBuilder()
			cmdSec.PutString(test.passwordIdent, "12345")

			t.Run("retrieve and test command object", func(t *testing.T) {
				cmd, err := bld.ArgumentObjectBuilder().Build()
				require.NoError(err)
				require.NotNil(cmd)
				testCommandsTree(t, cmd)
			})
		})

		t.Run("make results CUDs", func(t *testing.T) {
			cuds := bld.CUDBuilder()
			rec := cuds.Create(test.tablePhotos)
			rec.PutRecordID(appdef.SystemField_ID, test.tempPhotoID)
			rec.PutString(test.buyerIdent, test.buyerValue)
			rec.PutInt32(test.ageIdent, test.ageValue)
			rec.PutFloat32(test.heightIdent, test.heightValue)
			rec.PutBool(test.humanIdent, true)
			rec.PutBytes(test.photoIdent, test.photoValue)

			recRem := cuds.Create(test.tablePhotoRems)
			recRem.PutRecordID(appdef.SystemField_ID, test.tempRemarkID)
			recRem.PutRecordID(appdef.SystemField_ParentID, test.tempPhotoID)
			recRem.PutString(appdef.SystemField_Container, test.remarkIdent)
			recRem.PutRecordID(test.photoIdent, test.tempPhotoID)
			recRem.PutString(test.remarkIdent, test.remarkValue)
			recRem.PutString(test.emptinessIdent, test.emptinessValue)
		})

		t.Run("test build raw event", func(t *testing.T) {
			rawEvent, buildErr = bld.BuildRawEvent()
			require.NoError(buildErr)
			require.NotNil(rawEvent)
		})
	})

	t.Run("II. Save raw event to PLog & WLog and save Docs and CUDs demo", func(t *testing.T) {
		// 1. save to PLog
		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			require.True(rawID.IsRaw())
			switch rawID {
			case test.tempPhotoID:
				require.Equal(test.tablePhotos, t.QName())
				photoID = storageID
			case test.tempRemarkID:
				require.Equal(test.tablePhotoRems, t.QName())
				remarkID = storageID
			case test.tempSaleID:
				require.Equal(test.saleCmdDocName, t.QName())
				saleID = storageID
			case test.tempBasketID:
				require.Equal(appdef.NewQName(test.pkgName, test.basketIdent), t.QName())
				basketID = storageID
			case test.tempGoodsID[0]:
				require.Equal(appdef.NewQName(test.pkgName, test.goodIdent), t.QName())
				goodsID[0] = storageID
			case test.tempGoodsID[1]:
				require.Equal(appdef.NewQName(test.pkgName, test.goodIdent), t.QName())
				goodsID[1] = storageID
			}
			return nil
		},
		))
		require.NoError(saveErr)
		require.False(photoID.IsRaw())
		require.False(remarkID.IsRaw())
		require.False(saleID.IsRaw())
		require.False(basketID.IsRaw())
		require.False(goodsID[0].IsRaw())
		require.False(goodsID[1].IsRaw())
		defer pLogEvent.Release()

		testDbEvent(t, pLogEvent)
		require.Equal(test.workspace, pLogEvent.Workspace())
		require.Equal(test.wlogOfs, pLogEvent.WLogOffset())

		// 2. save to WLog
		err := app.Events().PutWlog(pLogEvent)
		require.NoError(err)

		// 3. save event command CUDs
		idP := istructs.NullRecordID
		idR := istructs.NullRecordID
		err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
			if r.QName() == test.tablePhotos {
				idP = r.ID()
				require.Equal(idP, photoID)
				testPhotoRow(t, r)
			}
			if r.QName() == test.tablePhotoRems {
				idR = r.ID()
				require.Equal(idR, remarkID)
				require.Equal(photoID, r.AsRecordID(test.photoIdent))
				require.Equal(test.remarkValue, r.AsString(test.remarkIdent))
				require.Equal(test.emptinessValue, r.AsString(test.emptinessIdent))
			}
		})
		require.NoError(err)
		require.NotEqual(istructs.NullRecordID, idP)
		require.NotEqual(istructs.NullRecordID, idR)
	})

	t.Run("III. Read event from PLog & PLog and reads CUD demo", func(t *testing.T) {

		t.Run("should be ok to read PLog", func(t *testing.T) {
			check := func(event istructs.IPLogEvent, err error) {
				require.NoError(err)
				require.NotNil(event)

				testDbEvent(t, event)
				require.Equal(test.workspace, event.Workspace())
				require.Equal(test.wlogOfs, event.WLogOffset())

				cmdRec := event.ArgumentObject().AsRecord()
				require.Equal(saleID, cmdRec.ID())
				require.Equal(test.buyerValue, cmdRec.AsString(test.buyerIdent))
			}

			t.Run("test single plog event reading", func(t *testing.T) {
				var event istructs.IPLogEvent
				err := app.Events().ReadPLog(context.Background(), test.partition, test.plogOfs, 1,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						require.Equal(test.plogOfs, plogOffset)
						event = ev
						return nil
					})
				defer event.Release()
				check(event, err)
			})

			t.Run("test sequential plog reading", func(t *testing.T) {
				cnt := 0
				err := app.Events().ReadPLog(context.Background(), test.partition, test.plogOfs, istructs.ReadToTheEnd,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						defer ev.Release()
						cnt++
						switch ev.QName() {
						case test.saleCmdName:
							require.Equal(test.plogOfs, plogOffset)
							check(ev, err)
						case test.changeCmdName:
							require.Equal(test.plogOfs+1, plogOffset)
						default:
							require.Fail("unexpected event in plog", "offset: %d, qname: %v", plogOffset, ev.QName())
						}
						return nil
					})
				require.NoError(err)
				require.Positive(cnt)
			})
		})

		t.Run("should be no events if read other PLog partition", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var event istructs.IPLogEvent
				err := app.Events().ReadPLog(context.Background(), test.partition+1, test.plogOfs, 1,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						require.Fail("should be no events if read other PLog partition")
						event = ev
						return nil
					})
				require.NoError(err)
				require.Nil(event)
			})

			t.Run("test sequential reading", func(t *testing.T) {
				var event istructs.IPLogEvent
				err := app.Events().ReadPLog(context.Background(), test.partition+1, test.plogOfs, istructs.ReadToTheEnd,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						require.Fail("should be no events if read other PLog partition")
						event = ev
						return nil
					})
				require.NoError(err)
				require.Nil(event)
			})
		})

		t.Run("should be ok to read WLog", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var event istructs.IWLogEvent
				err := app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs, 1,
					func(wlogOffset istructs.Offset, ev istructs.IWLogEvent) (err error) {
						require.Equal(test.wlogOfs, wlogOffset)
						event = ev
						return nil
					})

				require.NoError(err)
				require.NotNil(event)
				defer event.Release()
				testDbEvent(t, event)
			})

			t.Run("test sequential reading", func(t *testing.T) {
				var event istructs.IWLogEvent
				err := app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs, 1,
					func(wlogOffset istructs.Offset, ev istructs.IWLogEvent) (err error) {
						require.Equal(test.wlogOfs, wlogOffset)
						event = ev
						return nil
					})

				require.NoError(err)
				require.NotNil(event)
				defer event.Release()
				testDbEvent(t, event)
			})
		})

		t.Run("should be no event if read WLog from other WSID", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var wLogEvent istructs.IWLogEvent
				err := app.Events().ReadWLog(context.Background(), test.workspace+1, test.wlogOfs, 1,
					func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
						require.Fail("should be no event if read WLog from other WSID")
						return nil
					})
				require.NoError(err)
				require.Nil(wLogEvent)
			})

			t.Run("test sequential reading", func(t *testing.T) {
				var wLogEvent istructs.IWLogEvent
				err := app.Events().ReadWLog(context.Background(), test.workspace+1, test.wlogOfs, istructs.ReadToTheEnd,
					func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
						require.Fail("should be no event if read WLog from other WSID")
						return nil
					})
				require.NoError(err)
				require.Nil(wLogEvent)
			})
		})

		t.Run("read ODoc from IRecords must return NullRecord, see #17185", func(t *testing.T) {
			rec, err := app.Records().Get(test.workspace, true, saleID)
			require.NoError(err)
			require.NotNil(rec)
			require.Equal(appdef.NullQName, rec.QName())
		})

		t.Run("read CUDs from IRecords must return photo and remark records", func(t *testing.T) {
			rec, err := app.Records().Get(test.workspace, true, photoID)
			require.NoError(err)
			require.NotNil(rec)

			require.Equal(test.tablePhotos, rec.QName())
			testPhotoRow(t, rec)

			recRem, err := app.Records().Get(test.workspace, true, remarkID)
			require.NoError(err)
			require.NotNil(recRem)

			require.Equal(test.tablePhotoRems, recRem.QName())
			require.Equal(rec.ID(), recRem.AsRecordID(test.photoIdent))
			require.Equal(test.remarkValue, recRem.AsString(test.remarkIdent))
		})
	})

	var (
		changedHeights float32 = test.heightValue + 0.1
		changedPhoto           = []byte{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		changedRems    string  = "changes"
	)

	t.Run("VI. Build change event", func(t *testing.T) {

		// gets event builder
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs + 1,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs + 1,
					QName:             test.changeCmdName,
					RegisteredAt:      test.registeredTime + 1,
				},
			})

		t.Run("test build CUDs", func(t *testing.T) {
			cuds := bld.CUDBuilder()

			oldRec, err := app.Records().Get(test.workspace, true, photoID)
			require.NoError(err)
			require.NotNil(oldRec)

			rec := cuds.Update(oldRec)
			rec.PutFloat32(test.heightIdent, changedHeights) // +10 cm
			rec.PutBytes(test.photoIdent, changedPhoto)      // new photo

			oldRemRec, err := app.Records().Get(test.workspace, true, remarkID)
			require.NoError(err)
			require.NotNil(oldRec)

			remRec := cuds.Update(oldRemRec)
			remRec.PutString(test.remarkIdent, changedRems)
			remRec.PutString(test.emptinessIdent, "")
		})

		t.Run("test build raw event", func(t *testing.T) {
			rawEvent, buildErr = bld.BuildRawEvent()
			require.NoError(buildErr)
			require.NotNil(rawEvent)
		})
	})

	t.Run("VII. Save change event to PLog & WLog and save CUD demo", func(t *testing.T) {

		var pLogEvent istructs.IPLogEvent

		t.Run("test save to PLog", func(t *testing.T) {
			ev, saveErr := app.Events().PutPlog(rawEvent, buildErr, NewIDGenerator())
			require.NoError(saveErr)
			require.NotNil(ev)
			pLogEvent = ev
		})
		defer pLogEvent.Release()

		t.Run("test save to WLog", func(t *testing.T) {
			err := app.Events().PutWlog(pLogEvent)
			require.NoError(err)
		})

		t.Run("test apply PLog event records", func(t *testing.T) {
			err := app.Records().Apply(pLogEvent)
			require.NoError(err)
		})
	})

	t.Run("VIII. Read event from PLog & WLog and reads CUD", func(t *testing.T) {

		checkEvent := func(event istructs.IDbEvent, err error) {
			require.NoError(err)
			require.NotNil(event)

			t.Run("test PLog event CUDs", func(t *testing.T) {
				cudCount := 0
				for rec := range event.CUDs {
					if rec.QName() == test.tablePhotos {
						require.False(rec.IsNew())
						require.False(rec.IsActivated())
						require.False(rec.IsDeactivated())
						require.Equal(changedHeights, rec.AsFloat32(test.heightIdent))
						require.Equal(changedPhoto, rec.AsBytes(test.photoIdent))
					}
					if rec.QName() == test.tablePhotoRems {
						require.False(rec.IsNew())
						require.False(rec.IsActivated())
						require.False(rec.IsDeactivated())
						require.Equal(changedRems, rec.AsString(test.remarkIdent))
					}
					cudCount++
				}
				require.Equal(2, cudCount)
			})
		}

		t.Run("should be ok to read PLog", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var event istructs.IPLogEvent
				err := app.Events().ReadPLog(context.Background(), test.partition, test.plogOfs+1, 1,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						require.Equal(test.plogOfs+1, plogOffset)
						event = ev
						return nil
					})
				defer event.Release()
				checkEvent(event, err)
			})

			t.Run("test sequential reading", func(t *testing.T) {
				var event istructs.IPLogEvent
				err := app.Events().ReadPLog(context.Background(), test.partition, test.plogOfs+1, istructs.ReadToTheEnd,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						require.Equal(test.plogOfs+1, plogOffset)
						event = ev
						return nil
					})
				defer event.Release() // not necessary
				checkEvent(event, err)
			})
		})

		t.Run("should be ok to read WLog", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var event istructs.IWLogEvent
				err := app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs+1, 1,
					func(wlogOffset istructs.Offset, ev istructs.IWLogEvent) (err error) {
						require.Equal(test.wlogOfs+1, wlogOffset)
						event = ev
						return nil
					})
				defer event.Release()
				checkEvent(event, err)
			})

			t.Run("test sequential reading", func(t *testing.T) {
				var event istructs.IWLogEvent
				err := app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs+1, istructs.ReadToTheEnd,
					func(wlogOffset istructs.Offset, ev istructs.IWLogEvent) (err error) {
						require.Equal(test.wlogOfs+1, wlogOffset)
						event = ev
						return nil
					})
				defer event.Release() // not necessary
				checkEvent(event, err)
			})
		})

		t.Run("test read changed record", func(t *testing.T) {
			rec, err := app.Records().Get(test.workspace, true, photoID)
			require.NoError(err)
			require.NotNil(rec)

			require.Equal(test.tablePhotos, rec.QName())

			require.Equal(test.buyerValue, rec.AsString(test.buyerIdent))
			require.Equal(test.ageValue, rec.AsInt32(test.ageIdent))

			require.Equal(changedHeights, rec.AsFloat32(test.heightIdent))
			require.Equal(changedPhoto, rec.AsBytes(test.photoIdent))

			require.Equal(test.humanValue, rec.AsBool(test.humanIdent))

			recRem, err := app.Records().Get(test.workspace, true, remarkID)
			require.NoError(err)
			require.NotNil(recRem)

			require.Equal(test.tablePhotoRems, recRem.QName())
			require.Equal(changedRems, recRem.AsString(test.remarkIdent))
			require.Empty(recRem.AsString(test.emptinessIdent))
		})
	})

	t.Run("IX. Reread event from PLog and re-Apply CUDs", func(t *testing.T) {

		t.Run("restore photo record to previous value", func(t *testing.T) {
			rec, err := app.Records().Get(test.workspace, true, photoID)
			require.NoError(err)
			require.NotNil(rec)

			r := rec.(*recordType)
			require.Equal(changedHeights, rec.AsFloat32(test.heightIdent))
			r.PutFloat32(test.heightIdent, test.heightValue) // revert -10 cm
			require.Equal(changedPhoto, rec.AsBytes(test.photoIdent))
			r.PutBytes(test.photoIdent, test.photoValue) // revert to old photo
			require.NoError(r.build())

			// hack: use low level appRecordsType putRecord()
			bytes := r.storeToBytes()
			require.NotEmpty(bytes)
			err = app.Records().(*appRecordsType).putRecord(test.workspace, photoID, bytes)
			require.NoError(err)

			// check hack is success
			rec, err = app.Records().Get(test.workspace, true, photoID)
			require.NoError(err)
			require.NotNil(rec)
			require.Equal(test.heightValue, rec.AsFloat32(test.heightIdent))
		})

		t.Run("test reread PLog", func(t *testing.T) {
			var pLogEvent istructs.IPLogEvent
			err := app.Events().ReadPLog(context.Background(), test.partition, test.plogOfs+1, 1,
				func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
					require.Equal(test.plogOfs+1, plogOffset)
					pLogEvent = event
					return nil
				})
			require.NoError(err)
			require.NotNil(pLogEvent)
			defer pLogEvent.Release()

			checked := false
			for rec := range pLogEvent.CUDs {
				if rec.QName() == test.tablePhotos {
					require.False(rec.IsNew())
					require.False(rec.IsActivated())
					require.False(rec.IsDeactivated())
					require.Equal(changedHeights, rec.AsFloat32(test.heightIdent))
					require.Equal(changedPhoto, rec.AsBytes(test.photoIdent))
					checked = true
				}
			}

			require.True(checked)

			t.Run("test reApply CUDs", func(t *testing.T) {
				recCnt := 0
				err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
					switch r.QName() {
					case test.tablePhotos:
						require.Equal(changedHeights, r.AsFloat32(test.heightIdent))
						require.Equal(changedPhoto, r.AsBytes(test.photoIdent))
					case test.tablePhotoRems:
						require.Equal(changedRems, r.AsString(test.remarkIdent))
					default:
						require.FailNow("unexpected record QName from Apply2 to callback returned", "QName: «%v»", r.QName())
					}
					recCnt++
				})
				require.NoError(err)
				require.Equal(2, recCnt)
			})
		})

		t.Run("test rewritten record", func(t *testing.T) {
			rec, err := app.Records().Get(test.workspace, true, photoID)
			require.NoError(err)
			require.NotNil(rec)

			require.Equal(test.tablePhotos, rec.QName())

			require.Equal(test.buyerValue, rec.AsString(test.buyerIdent))
			require.Equal(test.ageValue, rec.AsInt32(test.ageIdent))
			require.Equal(changedHeights, rec.AsFloat32(test.heightIdent))
			require.Equal(changedPhoto, rec.AsBytes(test.photoIdent))

			require.Equal(test.humanValue, rec.AsBool(test.humanIdent))
		})
	})
}

func testCommandsTree(t *testing.T, cmd istructs.IObject) {
	require := require.New(t)
	test := test()

	t.Run("test command", func(t *testing.T) {
		require.NotNil(cmd)

		require.Equal(test.buyerValue, cmd.AsString(test.buyerIdent))
		require.Equal(test.ageValue, cmd.AsInt32(test.ageIdent))
		require.Equal(test.heightValue, cmd.AsFloat32(test.heightIdent))
		require.Equal(test.photoValue, cmd.AsBytes(test.photoIdent))

		require.Equal(test.humanValue, cmd.AsBool(test.humanIdent))
	})

	var basket istructs.IObject

	t.Run("test basket", func(t *testing.T) {
		var names []string
		for name := range cmd.Containers {
			names = append(names, name)
		}
		require.Len(names, 1)
		require.Equal(test.basketIdent, names[0])

		for c := range cmd.Children(test.basketIdent) {
			basket = c
			break
		}
		require.NotNil(basket)

		require.Equal(cmd.AsRecord().ID(), basket.AsRecord().Parent())
	})

	t.Run("test goods", func(t *testing.T) {
		var names []string
		for name := range basket.Containers {
			names = append(names, name)
		}
		require.Len(names, 1)
		require.Equal(test.goodIdent, names[0])

		var goods []istructs.IObject
		for g := range basket.Children(test.goodIdent) {
			goods = append(goods, g)
		}
		require.Len(goods, test.goodCount)

		for i := 0; i < test.goodCount; i++ {
			good := goods[i]
			require.Equal(basket.AsRecord().ID(), good.AsRecord().Parent())
			require.Equal(cmd.AsRecord().ID(), good.AsRecordID(test.saleIdent))
			require.Equal(test.goodNames[i], good.AsString(test.nameIdent))
			require.Equal(test.goodCodes[i], good.AsInt64(test.codeIdent))
			require.Equal(test.goodWeights[i], good.AsFloat64(test.weightIdent))
		}
	})
}

func testUnloggedObject(t *testing.T, cmd istructs.IObject) {
	require := require.New(t)
	test := test()

	hasPassword := false
	cmd.Fields(func(iField appdef.IField) bool {
		if hasPassword = iField.Name() == test.passwordIdent; hasPassword {
			return false
		}
		return true
	})
	require.True(hasPassword)

	require.Equal(maskString, cmd.AsString(test.passwordIdent))
}

func testPhotoRow(t *testing.T, photo istructs.IRowReader) {
	require := require.New(t)
	test := test()
	require.Equal(test.buyerValue, photo.AsString(test.buyerIdent))
	require.Equal(test.ageValue, photo.AsInt32(test.ageIdent))
	require.Equal(test.heightValue, photo.AsFloat32(test.heightIdent))
	require.Equal(test.photoValue, photo.AsBytes(test.photoIdent))
}

func testDbEvent(t *testing.T, event istructs.IDbEvent) {
	require := require.New(t)
	test := test()

	// test DBEvent event general entities
	require.Equal(test.saleCmdName, event.QName())
	require.Equal(test.registeredTime, event.RegisteredAt())
	require.True(event.Synced())
	require.Equal(test.device, event.DeviceID())
	require.Equal(test.syncTime, event.SyncedAt())

	// test DBEvent commands tree
	testCommandsTree(t, event.ArgumentObject())

	t.Run("test DBEvent CUDs", func(t *testing.T) {
		var cuds []istructs.IRowReader
		cnt := 0
		for row := range event.CUDs {
			cuds = append(cuds, row)
			if cnt == 0 {
				require.True(row.IsNew())
				require.Equal(test.tablePhotos, row.QName())
			}
			cnt++
		}
		require.Equal(2, cnt)
		require.Len(cuds, 2)
		testPhotoRow(t, cuds[0])
		require.Equal(cuds[0].AsRecordID(appdef.SystemField_ID), cuds[1].AsRecordID(test.photoIdent))
		require.Equal(test.remarkValue, cuds[1].AsString(test.remarkIdent))
	})
}

func Test_EventUpdateRawCud(t *testing.T) {
	// this test for https://dev.heeus.io/launchpad/#!25853
	require := require.New(t)

	appName := istructs.AppQName_test1_app1
	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "cDoc")
	recName := appdef.NewQName("test", "cRec")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	t.Run("should be ok to construct application", func(t *testing.T) {
		ws := adb.AddWorkspace(wsName)

		doc := ws.AddCDoc(docName)
		doc.AddField("new", appdef.DataKind_bool, true)
		doc.AddField("rec", appdef.DataKind_RecordID, false)
		doc.AddField("emptied", appdef.DataKind_string, false)
		doc.AddContainer("rec", recName, 0, 1)

		rec := ws.AddCRecord(recName)
		rec.AddField("data", appdef.DataKind_string, false)
	})

	cfgs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		return cfgs
	}()

	const (
		simpleTest uint = iota
		retryTest

		testCount
	)

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	ws := istructs.WSID(1)

	expectedQName := docName
	idGenerator := NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
		require.EqualValues(1, rawID)
		require.EqualValues(expectedQName, t.QName())
		return nil
	})

	for test := simpleTest; test < testCount*2; test += 2 { // test - docID, test+1 - recID

		app, err := provider.BuiltIn(appName)
		require.NoError(err)

		docID := istructs.NewCDocCRecordID(istructs.FirstBaseRecordID + istructs.RecordID(test))

		t.Run("should be ok to create CDoc", func(t *testing.T) {
			bld := app.Events().GetNewRawEventBuilder(
				istructs.NewRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 1,
						PLogOffset:        istructs.Offset(100500 + test),
						Workspace:         ws,
						WLogOffset:        istructs.Offset(100500 + test),
						QName:             istructs.QNameCommandCUD, // sys.CUD
						RegisteredAt:      1,
					},
				})

			create := bld.CUDBuilder().Create(docName)
			create.PutRecordID(appdef.SystemField_ID, 1)
			create.PutBool("new", true)
			create.PutString("emptied", "to be emptied")

			rawEvent, err := bld.BuildRawEvent()
			require.NoError(err)
			require.NotNil(rawEvent)

			expectedQName = docName
			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err, idGenerator)
			require.NotNil(pLogEvent)
			require.NoError(saveErr)
			require.True(pLogEvent.Error().ValidEvent())

			t.Run("should be ok to apply CDoc records", func(t *testing.T) {
				recCnt := 0
				err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
					require.EqualValues(docName, r.QName())
					require.EqualValues(docID, r.ID())
					require.Zero(r.AsRecordID("rec"))
					recCnt++
				})
				require.Equal(1, recCnt)
			})
		})

		recID := docID + 1

		t.Run("should be ok to update CDoc", func(t *testing.T) {
			bld := app.Events().GetNewRawEventBuilder(
				istructs.NewRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 1,
						PLogOffset:        istructs.Offset(100501 + test),
						Workspace:         ws,
						WLogOffset:        istructs.Offset(100501 + test),
						QName:             istructs.QNameCommandCUD, // sys.CUD
						RegisteredAt:      1,
					},
				})

			create := bld.CUDBuilder().Create(recName)
			create.PutRecordID(appdef.SystemField_ID, 1)
			create.PutRecordID(appdef.SystemField_ParentID, docID)
			create.PutString(appdef.SystemField_Container, "rec")
			create.PutString("data", "test data")

			update := bld.CUDBuilder().Update(
				func() istructs.IRecord {
					rec, err := app.Records().Get(ws, true, docID)
					require.NoError(err)
					return rec
				}())
			update.PutBool("new", false)
			update.PutRecordID("rec", 1)
			update.PutString("emptied", "")

			rawEvent, err := bld.BuildRawEvent()
			require.NoError(err)
			require.NotNil(rawEvent)

			expectedQName = recName
			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err, idGenerator)
			require.NotNil(pLogEvent)
			require.NoError(saveErr)
			require.True(pLogEvent.Error().ValidEvent())

			switch test {
			case retryTest:
				t.Run("should be ok to reread PLog event", func(t *testing.T) {
					pLogEvent.Release()

					pLogEvent = nil
					err := app.Events().ReadPLog(context.Background(), 1, istructs.Offset(100501+test), 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
						require.EqualValues(100501+test, plogOffset)
						pLogEvent = event
						return nil
					})
					require.NoError(err)
					require.NotNil(pLogEvent)
					require.True(pLogEvent.Error().ValidEvent())
				})
			}

			t.Run("should be ok to apply CDoc records", func(t *testing.T) {
				recCnt := 0
				err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
					switch id := r.ID(); id {
					case docID:
						require.EqualValues(docName, r.QName())
						require.EqualValues(recID, r.AsRecordID("rec"), "error #25853 here!")
					case recID:
						require.EqualValues(recName, r.QName())
						require.EqualValues(docID, r.Parent())
						require.EqualValues("rec", r.Container())
						require.EqualValues("test data", r.AsString("data"))
					default:
						require.Fail("unexpected record applied")
					}
					recCnt++
				})
				require.Equal(2, recCnt)
			})

			pLogEvent.Release()

			t.Run("should be ok to reread CDoc record", func(t *testing.T) {
				rec, err := app.Records().Get(ws, true, docID)
				require.NoError(err)
				require.EqualValues(docName, rec.QName())
				require.EqualValues(rec.AsRecordID("rec"), recID, "error #25853 here!")
			})
		})

	}
}

func Test_UpdateCorrupted(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1
	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	t.Run("should be ok to build AppDef", func(t *testing.T) {
		ws := adb.AddWorkspace(wsName)
		doc := ws.AddCDoc(docName)
		doc.SetSingleton()
		doc.AddField("option", appdef.DataKind_int64, true)

		_ = adb.MustBuild()
	})

	cfgs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		return cfgs
	}()

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	t.Run("should be ok to put new sys.CUD event", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 1,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        100500,
					QName:             istructs.QNameCommandCUD, // sys.CUD
					RegisteredAt:      1,
				},
			})

		cud := bld.CUDBuilder().Create(docName)
		cud.PutRecordID(appdef.SystemField_ID, 1)
		cud.PutInt64("option", 8)

		rawEvent, err := bld.BuildRawEvent()
		require.NoError(err)
		require.NotNil(rawEvent)

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			return errors.New("unexpected call ID generator from singleton CDoc creation")
		}))
		require.NotNil(pLogEvent)
		require.NoError(saveErr)
		require.True(pLogEvent.Error().ValidEvent())
		require.Equal(pLogEvent.QName(), istructs.QNameCommandCUD)

		pLogEvent.Release()
	})

	var origEventBytes []byte = nil

	t.Run("should ok to read PLog event", func(t *testing.T) {
		var pLogEvent istructs.IPLogEvent
		err := app.Events().ReadPLog(context.Background(), 1, istructs.Offset(100500), 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			require.EqualValues(100500, plogOffset)
			pLogEvent = event
			return nil
		})
		require.NoError(err)
		require.NotNil(pLogEvent)
		require.True(pLogEvent.Error().ValidEvent())
		require.Equal(pLogEvent.QName(), istructs.QNameCommandCUD)

		origEventBytes = utils.CopyBytes(pLogEvent.Bytes())
		require.NotEmpty(origEventBytes)

		pLogEvent.Release()
	})

	require.NotNil(origEventBytes)

	t.Run("should be ok to update corrupted event", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					EventBytes:        utils.CopyBytes(origEventBytes),
					HandlingPartition: 1,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        100500,
					QName:             istructs.QNameForCorruptedData, // sys.Corrupted
					RegisteredAt:      1,
				},
			})

		rawEvent, err := bld.BuildRawEvent()
		require.NoError(err)
		require.NotNil(rawEvent)

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			return errors.New("unexpected call ID generator from update corrupted event")
		}))
		require.NotNil(pLogEvent)
		require.NoError(saveErr)

		require.Equal(pLogEvent.QName(), istructs.QNameForCorruptedData)
		require.False(pLogEvent.Error().ValidEvent())
		require.EqualValues(pLogEvent.Error().OriginalEventBytes(), origEventBytes)

		pLogEvent.Release()
	})

	t.Run("should be ok to reread corrupted PLog event", func(t *testing.T) {
		var pLogEvent istructs.IPLogEvent
		err := app.Events().ReadPLog(context.Background(), 1, istructs.Offset(100500), 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			require.EqualValues(100500, plogOffset)
			pLogEvent = event
			return nil
		})
		require.NotNil(pLogEvent)
		require.NoError(err)

		require.Equal(pLogEvent.QName(), istructs.QNameForCorruptedData)
		require.False(pLogEvent.Error().ValidEvent())
		require.EqualValues(pLogEvent.Error().OriginalEventBytes(), origEventBytes)

		pLogEvent.Release()
	})
}

func Test_BuildPLogEvent(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1
	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	t.Run("should be ok to build AppDef", func(t *testing.T) {
		ws := adb.AddWorkspace(wsName)
		doc := ws.AddCDoc(docName)
		doc.SetSingleton()
		doc.AddField("option", appdef.DataKind_int64, true)

		_ = adb.MustBuild()
	})

	cfgs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		return cfgs
	}()

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	t.Run("should be ok to put new sys.CUD event", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 1,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        100500,
					QName:             istructs.QNameCommandCUD, // sys.CUD
					RegisteredAt:      1,
				},
			})

		cud := bld.CUDBuilder().Create(docName)
		cud.PutRecordID(appdef.SystemField_ID, 1)
		cud.PutInt64("option", 8)

		rawEvent, err := bld.BuildRawEvent()
		require.NoError(err)
		require.NotNil(rawEvent)

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			return errors.New("unexpected call ID generator from singleton CDoc creation")
		}))
		require.NotNil(pLogEvent)
		require.NoError(saveErr)
		require.True(pLogEvent.Error().ValidEvent())
		require.Equal(pLogEvent.QName(), istructs.QNameCommandCUD)

		pLogEvent.Release()
	})

	var origEventBytes []byte = nil

	t.Run("should ok to read PLog event", func(t *testing.T) {
		var pLogEvent istructs.IPLogEvent
		err := app.Events().ReadPLog(context.Background(), 1, istructs.Offset(100500), 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			require.EqualValues(100500, plogOffset)
			pLogEvent = event
			return nil
		})
		require.NoError(err)
		require.NotNil(pLogEvent)
		require.True(pLogEvent.Error().ValidEvent())
		require.Equal(pLogEvent.QName(), istructs.QNameCommandCUD)

		origEventBytes = utils.CopyBytes(pLogEvent.Bytes())
		require.NotEmpty(origEventBytes)

		pLogEvent.Release()
	})

	require.NotNil(origEventBytes)

	t.Run("should be ok to build PLog corrupted event", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					EventBytes:        utils.CopyBytes(origEventBytes),
					HandlingPartition: 1,
					PLogOffset:        istructs.NullOffset,
					Workspace:         1,
					WLogOffset:        100500,
					QName:             istructs.QNameForCorruptedData, // sys.Corrupted
					RegisteredAt:      1,
				},
			})

		rawEvent, err := bld.BuildRawEvent()
		require.NoError(err)
		require.NotNil(rawEvent)

		pLogEvent := app.Events().BuildPLogEvent(rawEvent)
		require.NotNil(pLogEvent)

		pLogEvent.Release()

		require.Equal(pLogEvent.QName(), istructs.QNameForCorruptedData)
		require.False(pLogEvent.Error().ValidEvent())
		require.EqualValues(pLogEvent.Error().OriginalEventBytes(), origEventBytes)
		require.EqualValues(100500, pLogEvent.WLogOffset())

		t.Run("should be ok to put PLog event into WLog", func(t *testing.T) {
			err := app.Events().PutWlog(pLogEvent)
			require.NoError(err)
		})
	})

	t.Run("should be ok to reread corrupted WLog event", func(t *testing.T) {
		var wLogEvent istructs.IWLogEvent
		err := app.Events().ReadWLog(context.Background(), 1, istructs.Offset(100500), 1, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) error {
			require.EqualValues(100500, wlogOffset)
			wLogEvent = event
			return nil
		})
		require.NotNil(wLogEvent)
		require.NoError(err)

		require.Equal(wLogEvent.QName(), istructs.QNameForCorruptedData)
		require.False(wLogEvent.Error().ValidEvent())
		require.EqualValues(wLogEvent.Error().OriginalEventBytes(), origEventBytes)

		wLogEvent.Release()
	})

	t.Run("test panics while build PLog corrupted event", func(t *testing.T) {

		t.Run("should panic if not sys.Corrupted raw event", func(t *testing.T) {
			bld := app.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						EventBytes:        utils.CopyBytes(origEventBytes),
						HandlingPartition: 1,
						PLogOffset:        istructs.NullOffset,
						Workspace:         1,
						WLogOffset:        100500,
						QName:             istructs.QNameCommandCUD, // <- error here
						RegisteredAt:      1,
					},
				})

			cud := bld.CUDBuilder().Create(docName)
			cud.PutRecordID(appdef.SystemField_ID, 1)
			cud.PutInt64("option", 8)

			rawEvent, err := bld.BuildRawEvent()
			require.NoError(err)
			require.NotNil(rawEvent)

			require.PanicsWith(
				func() { app.Events().BuildPLogEvent(rawEvent) },
				require.Is(ErrorEventNotValidError),
				require.Has(istructs.QNameCommandCUD.String()),
			)
		})

		t.Run("should panic if not null PLog offset", func(t *testing.T) {
			bld := app.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						EventBytes:        utils.CopyBytes(origEventBytes),
						HandlingPartition: 1,
						PLogOffset:        100501, // <- error here
						Workspace:         1,
						WLogOffset:        100500,
						QName:             istructs.QNameForCorruptedData,
						RegisteredAt:      1,
					},
				})

			cud := bld.CUDBuilder().Create(docName)
			cud.PutRecordID(appdef.SystemField_ID, 1)
			cud.PutInt64("option", 8)

			rawEvent, err := bld.BuildRawEvent()
			require.NoError(err)
			require.NotNil(rawEvent)

			require.PanicsWith(
				func() { app.Events().BuildPLogEvent(rawEvent) },
				require.Is(ErrorEventNotValidError),
				require.Has("100501"),
			)
		})
	})
}

func Test_SingletonCDocEvent(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1
	wsName := appdef.NewQName("test", "workspace")
	docName, doc2Name := appdef.NewQName("test", "cDoc"), appdef.NewQName("test", "cDoc2")
	docID := istructs.NullRecordID

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	t.Run("should be ok to construct singleton CDoc", func(t *testing.T) {
		ws := adb.AddWorkspace(wsName)

		doc := ws.AddCDoc(docName)
		doc.SetSingleton()
		doc.AddField("option", appdef.DataKind_int64, true)

		doc2 := ws.AddCDoc(doc2Name)
		doc2.SetSingleton()
		doc2.AddField("option", appdef.DataKind_int64, true)
	})

	cfgs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		return cfgs
	}()

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	docID, err = cfgs.GetConfig(appName).singletons.ID(docName)
	require.NoError(err)

	t.Run("should be ok to read not created singleton CDoc by QName", func(t *testing.T) {
		rec, err := app.Records().GetSingleton(1, docName)
		require.NoError(err)
		require.Equal(appdef.NullQName, rec.QName())
		require.Equal(docID, rec.ID())
	})

	t.Run("should be ok to create singleton CDoc", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 1,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        100500,
					QName:             istructs.QNameCommandCUD, // sys.CUD
					RegisteredAt:      1,
				},
			})

		cud := bld.CUDBuilder().Create(docName)
		cud.PutRecordID(appdef.SystemField_ID, 1)
		cud.PutInt64("option", 8)

		rawEvent, err := bld.BuildRawEvent()
		require.NoError(err)
		require.NotNil(rawEvent)

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			return errors.New("unexpected call ID generator from singleton CDoc creation")
		}))
		require.NotNil(pLogEvent)
		require.NoError(saveErr)
		require.True(pLogEvent.Error().ValidEvent())

		t.Run("newly created singleton CDoc must be ok", func(t *testing.T) {
			recCnt := 0
			for rec := range pLogEvent.CUDs {
				require.Equal(docName, rec.QName())
				require.Equal(docID, rec.ID())
				require.True(rec.IsNew())
				require.Equal(int64(8), rec.AsInt64("option"))
				recCnt++
			}
			require.Equal(1, recCnt)
		})

		t.Run("should be ok to apply singleton CDoc records", func(t *testing.T) {
			recCnt := 0
			err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
				require.Equal(docName, r.QName())
				require.Equal(docID, r.ID())
				require.Equal(int64(8), r.AsInt64("option"))
				recCnt++
			})
			require.Equal(1, recCnt)
		})
	})

	t.Run("should be ok to read singleton CDoc by QName", func(t *testing.T) {
		rec, err := app.Records().GetSingleton(1, docName)
		require.NoError(err)
		require.Equal(docName, rec.QName())
		require.Equal(docID, rec.ID())
		require.Equal(int64(8), rec.AsInt64("option"))
	})

	t.Run("must fail to read singleton CDoc by unknown QName", func(t *testing.T) {
		rec, err := app.Records().GetSingleton(1, appdef.NewQName("test", "unknownCDoc"))
		require.ErrorIs(err, singletons.ErrNameNotFound)
		require.Equal(appdef.NullQName, rec.QName())
		require.Equal(istructs.NullRecordID, rec.ID())
	})

	t.Run("must fail to attempt singleton CDoc recreation", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 1,
					PLogOffset:        100501,
					Workspace:         1,
					WLogOffset:        100501,
					QName:             istructs.QNameCommandCUD, // sys.CUD
					RegisteredAt:      1,
				},
			})

		cud := bld.CUDBuilder().Create(docName)
		cud.PutRecordID(appdef.SystemField_ID, 1)
		cud.PutInt64("option", 88)

		rawEvent, buildErr := bld.BuildRawEvent()
		require.NotNil(rawEvent)
		require.Error(buildErr, require.Is(ErrRecordIDUniqueViolationError), require.Has(cud))

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			return errors.New("unexpected call ID generator from singleton CDoc creation")
		}))
		require.NotNil(pLogEvent)
		require.NoError(saveErr)
		require.False(pLogEvent.Error().ValidEvent())

		require.Panics(
			func() {
				_ = app.Records().Apply2(pLogEvent, func(_ istructs.IRecord) {})
			},
			require.Is(ErrorEventNotValidError), require.Has(buildErr))
	})

	t.Run("must fail to repeatedly create singleton CDoc", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 1,
					PLogOffset:        100501,
					Workspace:         1,
					WLogOffset:        100501,
					QName:             istructs.QNameCommandCUD, // sys.CUD
					RegisteredAt:      1,
				},
			})

		for i := 1; i <= 2; i++ {
			cud := bld.CUDBuilder().Create(doc2Name)
			cud.PutRecordID(appdef.SystemField_ID, istructs.RecordID(i))
			cud.PutInt64("option", 88)
		}

		rawEvent, buildErr := bld.BuildRawEvent()
		require.NotNil(rawEvent)
		require.Error(buildErr, require.Is(ErrRecordIDUniqueViolationError),
			require.Has(doc2Name))
	})

	t.Run("should be ok to update singleton CDoc", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 1,
					PLogOffset:        100502,
					Workspace:         1,
					WLogOffset:        100502,
					QName:             istructs.QNameCommandCUD, // sys.CUD
					RegisteredAt:      2,
				},
			})

		cud := bld.CUDBuilder().Update(
			func() istructs.IRecord {
				rec, err := app.Records().Get(1, true, docID)
				require.NoError(err)
				return rec
			}())
		cud.PutInt64("option", 888)

		rawEvent, err := bld.BuildRawEvent()
		require.NoError(err)
		require.NotNil(rawEvent)

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			return errors.New("unexpected call ID generator while singleton CDoc update")
		}))
		require.NotNil(pLogEvent)
		require.NoError(saveErr)
		require.True(pLogEvent.Error().ValidEvent())

		t.Run("updated singleton CDoc must be ok", func(t *testing.T) {
			recCnt := 0
			for rec := range pLogEvent.CUDs {
				require.Equal(docName, rec.QName())
				require.Equal(docID, rec.ID())
				require.False(rec.IsNew())
				require.Equal(int64(888), rec.AsInt64("option"))
				recCnt++
			}
			require.Equal(1, recCnt)
		})

		t.Run("should be ok to apply singleton CDoc update records", func(t *testing.T) {
			recCnt := 0
			err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
				require.Equal(docName, r.QName())
				require.Equal(docID, r.ID())
				require.Equal(int64(888), r.AsInt64("option"))
				recCnt++
			})
			require.Equal(1, recCnt)
		})

		t.Run("should be ok to read updated singleton CDoc from records", func(t *testing.T) {
			rec, err := app.Records().Get(1, true, docID)
			require.NoError(err)
			require.Equal(docName, rec.QName())
			require.Equal(docID, rec.ID())
			require.Equal(int64(888), rec.AsInt64("option"))
		})
	})
}

func TestEventBuild_Error(t *testing.T) {
	require := require.New(t)
	test := test()

	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(test.appName)
	require.NoError(err)

	var rawEvent istructs.IRawEvent
	var buildErr error

	eventBuilder := func(cmd appdef.QName) istructs.IRawEventBuilder {
		return app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             cmd,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
	}

	t.Run("Build null-name event must have error", func(t *testing.T) {
		bld := eventBuilder(appdef.NullQName)
		rawEvent, buildErr = bld.BuildRawEvent()
		require.Error(buildErr)
		require.NotNil(rawEvent)
	})

	t.Run("Build sys.CUD must have error if empty CUDs", func(t *testing.T) {
		bld := eventBuilder(istructs.QNameCommandCUD)
		rawEvent, buildErr = bld.BuildRawEvent()
		require.Error(buildErr, require.Is(ErrCUDsMissedError), require.Has(istructs.QNameCommandCUD))
		require.NotNil(rawEvent)
	})

	t.Run("Build invalid name command name must have error", func(t *testing.T) {
		bld := eventBuilder(appdef.NewQName("unknown", "command-name"))
		rawEvent, buildErr = bld.BuildRawEvent()
		require.Error(buildErr)
		require.NotNil(rawEvent)
	})

	t.Run("Error in ArgumentObject", func(t *testing.T) {
		bld := eventBuilder(test.saleCmdName)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
		cmd.PutFloat32(test.buyerIdent, 123.321)

		rawEvent, buildErr = bld.BuildRawEvent()
		require.Error(buildErr)
		require.NotNil(rawEvent)
	})

	t.Run("Error if error in nested child of ArgumentObject", func(t *testing.T) {
		bld := eventBuilder(test.saleCmdName)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		basket := cmd.ChildBuilder(test.basketIdent)
		basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)
		good := basket.ChildBuilder(test.goodIdent)
		good.PutRecordID(appdef.SystemField_ID, test.tempGoodsID[0])
		good.PutBytes("unknownField", []byte{1, 2})

		rawEvent, buildErr = bld.BuildRawEvent()
		require.Error(buildErr)
		require.NotNil(rawEvent)
	})

	t.Run("Error in ArgumentUnloggedObject", func(t *testing.T) {
		bld := eventBuilder(test.saleCmdName)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		basket := cmd.ChildBuilder(test.basketIdent)
		basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)

		cmdSec := bld.ArgumentUnloggedObjectBuilder()
		cmdSec.PutInt64(test.passwordIdent, 12345)

		rawEvent, buildErr = bld.BuildRawEvent()
		require.Error(buildErr)
		require.NotNil(rawEvent)
	})

	t.Run("Error in CUD (creates)", func(t *testing.T) {
		bld := eventBuilder(test.saleCmdName)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		basket := cmd.ChildBuilder(test.basketIdent)
		basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)

		cmdSec := bld.ArgumentUnloggedObjectBuilder()
		cmdSec.PutString(test.passwordIdent, "12345")

		cuds := bld.CUDBuilder()
		cud := cuds.Create(test.tablePhotoRems)
		cud.PutBytes("unknownField", []byte{1, 2})

		rawEvent, buildErr = bld.BuildRawEvent()
		require.Error(buildErr)
		require.NotNil(rawEvent)
	})

	t.Run("Error in CUD (updates)", func(t *testing.T) {

		getPhoto := func() istructs.IRecord {
			r := newRecord(test.AppCfg)
			r.PutQName(appdef.SystemField_QName, test.tablePhotos)
			r.PutRecordID(appdef.SystemField_ID, 100500)
			r.PutString(test.buyerIdent, test.buyerValue)
			err := r.build()
			require.NoError(err)
			return r
		}

		getPhotoRem := func() istructs.IRecord {
			r := newRecord(test.AppCfg)
			r.PutQName(appdef.SystemField_QName, test.tablePhotoRems)
			r.PutRecordID(appdef.SystemField_ID, 100501)
			r.PutRecordID(appdef.SystemField_ParentID, 100500)
			r.PutString(appdef.SystemField_Container, test.remarkIdent)
			r.PutRecordID(test.photoIdent, 100500)
			r.PutString(test.remarkIdent, test.remarkValue)
			err := r.build()
			require.NoError(err)
			return r
		}

		t.Run("prepare exists photo records", func(t *testing.T) {
			rec := getPhoto().(*recordType)
			data := rec.storeToBytes()
			err := app.Records().(*appRecordsType).putRecord(test.workspace, rec.id, data)
			require.NoError(err)

			rec = getPhotoRem().(*recordType)
			data = rec.storeToBytes()
			err = app.Records().(*appRecordsType).putRecord(test.workspace, rec.id, data)
			require.NoError(err)
		})

		t.Run("update not applicable by QName", func(t *testing.T) {
			bld := eventBuilder(test.changeCmdName)

			rec := getPhoto()

			cud := bld.CUDBuilder().Update(rec)
			cud.PutQName(appdef.SystemField_QName, test.tablePhotoRems) // <- error here
			cud.PutRecordID(appdef.SystemField_ID, 100501)
			cud.PutRecordID(appdef.SystemField_ParentID, 100500)
			cud.PutString(appdef.SystemField_Container, test.remarkIdent)
			cud.PutRecordID(test.photoIdent, 100500)
			cud.PutString(test.remarkIdent, test.remarkValue)

			_, buildErr = bld.BuildRawEvent()
			require.Error(buildErr, require.Is(ErrUnableToUpdateSystemFieldError), require.HasAll(rec, appdef.SystemField_QName))
		})

		t.Run("update unknown field", func(t *testing.T) {
			bld := eventBuilder(test.changeCmdName)

			rec := getPhoto()

			cud := bld.CUDBuilder().Update(rec)
			cud.PutFloat32("unknownField", 7.7)

			_, buildErr = bld.BuildRawEvent()
			require.Error(buildErr, require.Is(ErrNameNotFoundError), require.Has("unknownField"))
		})

		t.Run("can`t change system fields", func(t *testing.T) {

			t.Run("can`t change sys.ID", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutRecordID(appdef.SystemField_ID, 100502)
				_, buildErr = bld.BuildRawEvent()
				require.Error(buildErr, require.Is(ErrUnableToUpdateSystemFieldError,
					require.HasAll(getPhotoRem(), appdef.SystemField_ID)))
			})

			t.Run("can`t change sys.ParentID", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutRecordID(appdef.SystemField_ParentID, 100502)
				_, buildErr = bld.BuildRawEvent()
				require.Error(buildErr, require.Is(ErrUnableToUpdateSystemFieldError,
					require.HasAll(getPhotoRem(), appdef.SystemField_ParentID)))
			})

			t.Run("can`t change sys.Container", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutString(appdef.SystemField_Container, test.basketIdent) // error here
				_, buildErr = bld.BuildRawEvent()
				require.Error(buildErr, require.Is(ErrUnableToUpdateSystemFieldError,
					require.HasAll(getPhotoRem(), appdef.SystemField_Container)))
			})

			t.Run("allow to change sys.IsActive", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutBool(appdef.SystemField_IsActive, false)
				rawEvent, buildErr = bld.BuildRawEvent()
				require.NoError(buildErr)
				require.NotNil(rawEvent)
			})
		})

		t.Run("update has unknown field", func(t *testing.T) {
			bld := eventBuilder(test.changeCmdName)

			rec := getPhoto()

			cud := bld.CUDBuilder().Update(rec)
			cud.PutString("unknown", "someValue")

			rawEvent, buildErr = bld.BuildRawEvent()
			require.Error(buildErr, require.Is(ErrNameNotFoundError), require.Has("unknown"))
			require.NotNil(rawEvent)
		})

	})

	t.Run("Errors in Generate ID", func(t *testing.T) {

		t.Run("Error in Generate ArgumentObject Object ID", func(t *testing.T) {
			bld := eventBuilder(test.saleCmdName)

			cmd := bld.ArgumentObjectBuilder()
			cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
			cmd.PutString(test.buyerIdent, test.buyerValue)
			basket := cmd.ChildBuilder(test.basketIdent)
			basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)

			cmdSec := bld.ArgumentUnloggedObjectBuilder()
			cmdSec.PutString(test.passwordIdent, "12345")

			rawEvent, buildErr = bld.BuildRawEvent()
			require.NoError(buildErr)
			require.NotNil(rawEvent)

			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
				if rawID == test.tempBasketID {
					require.Equal(appdef.NewQName(test.pkgName, test.basketIdent), t.QName())
					return ErrWrongRecordID("test error")
				}
				return nil
			}))
			require.False(pLogEvent.Error().ValidEvent())
			require.Contains(pLogEvent.Error().ErrStr(), ErrWrongRecordIDError.Error())
			require.NoError(saveErr)
			require.NotNil(pLogEvent)
		})

		t.Run("Error in Generate CUD ID", func(t *testing.T) {
			bld := eventBuilder(test.changeCmdName)

			cud := bld.CUDBuilder().Create(test.tablePhotos)
			cud.PutRecordID(appdef.SystemField_ID, 100500)
			cud.PutString(test.buyerIdent, test.buyerValue)

			cud = bld.CUDBuilder().Create(test.tablePhotoRems)
			cud.PutRecordID(appdef.SystemField_ID, 7)
			cud.PutRecordID(appdef.SystemField_ParentID, 100500)
			cud.PutString(appdef.SystemField_Container, test.remarkIdent)
			cud.PutRecordID(test.photoIdent, 100500)
			cud.PutString(test.remarkIdent, test.remarkValue)

			rawEvent, buildErr = bld.BuildRawEvent()
			require.NoError(buildErr)
			require.NotNil(rawEvent)

			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr, NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
				if rawID == 7 {
					require.Equal(test.tablePhotoRems, t.QName())
					return ErrWrongRecordID("test error")
				}
				return nil
			}))
			require.False(pLogEvent.Error().ValidEvent())
			require.Contains(pLogEvent.Error().ErrStr(), ErrWrongRecordIDError.Error())
			require.NoError(saveErr)
			require.NotNil(pLogEvent)

			require.Panics(
				func() {
					_ = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {})
				},
				require.Is(ErrorEventNotValidError), require.Has(ErrWrongRecordIDError))
		})
	})
}

func Test_LoadStoreEvent_Bytes(t *testing.T) {
	require := require.New(t)

	ev1 := newTestEvent(100500, 500)
	testDbEvent(t, ev1)

	ev1.argUnlObj.maskValues()

	b := ev1.storeToBytes()

	ev2 := newEmptyTestEvent()
	err := ev2.loadFromBytes(b)
	require.NoError(err)

	require.Equal(istructs.Offset(100500), ev2.pLogOffs)
	require.Equal(istructs.Offset(500), ev2.wLogOffs)

	testDbEvent(t, ev2)
	testUnloggedObject(t, ev2.ArgumentUnloggedObject())

	// #2785
	t.Run("should be supports emptied fields in CUDs", func(t *testing.T) {
		test := test()

		emptiedPhotoID := test.tempPhotoID + 1
		emptiedRemarkID := test.tempRemarkID + 1
		ev1 := newTestEvent(100500, 500)

		t.Run("put CUD with emptied photo", func(t *testing.T) {
			cud := ev1.CUDBuilder().Create(test.tablePhotos)
			cud.PutRecordID(appdef.SystemField_ID, emptiedPhotoID)
			cud.PutString(test.buyerIdent, "") // empty here, but next filled
			cud.PutString(test.buyerIdent, test.buyerValue)
			cud.PutInt32(test.ageIdent, test.ageValue)
			cud.PutBytes(test.photoIdent, nil) // empty bytes-field
		})

		t.Run("put CUD with emptied photo remark", func(t *testing.T) {
			cud := ev1.CUDBuilder().Create(test.tablePhotoRems)
			cud.PutRecordID(appdef.SystemField_ID, emptiedRemarkID)
			cud.PutRecordID(appdef.SystemField_ParentID, emptiedPhotoID)
			cud.PutString(appdef.SystemField_Container, test.remarkIdent)
			cud.PutRecordID(test.photoIdent, test.tempPhotoID)
			cud.PutString(test.remarkIdent, "") // empty string-field
		})

		b := ev1.storeToBytes()

		ev2 := newEmptyTestEvent()
		err := ev2.loadFromBytes(b)
		require.NoError(err)

		t.Run("should ok to find CUDs with emptied field", func(t *testing.T) {
			for cud := range ev2.CUDs {
				switch cud.AsRecordID(appdef.SystemField_ID) {
				case emptiedPhotoID:
					fields := make(map[appdef.FieldName]interface{})
					for fld, val := range cud.SpecifiedValues {
						fields[fld.Name()] = val
					}
					require.Equal(
						map[appdef.FieldName]interface{}{
							test.buyerIdent:             test.buyerValue,
							test.ageIdent:               test.ageValue,
							test.photoIdent:             []byte{}, // emptied bytes-field
							appdef.SystemField_ID:       emptiedPhotoID,
							appdef.SystemField_IsActive: true,
							appdef.SystemField_QName:    test.tablePhotos,
						},
						fields)
				case emptiedRemarkID:
					fields := make(map[appdef.FieldName]interface{})
					for fld, val := range cud.SpecifiedValues {
						fields[fld.Name()] = val
					}
					require.Equal(
						map[appdef.FieldName]interface{}{
							test.photoIdent:              test.tempPhotoID,
							test.remarkIdent:             "", // emptied string-field
							appdef.SystemField_ID:        emptiedRemarkID,
							appdef.SystemField_IsActive:  true,
							appdef.SystemField_QName:     test.tablePhotoRems,
							appdef.SystemField_Container: test.remarkIdent,
							appdef.SystemField_ParentID:  emptiedPhotoID,
						},
						fields)
				}
			}
		})
	})
}

func Test_LoadEvent_DamagedBytes(t *testing.T) {
	require := require.New(t)

	ev1 := newTestEvent(100500, 500)

	// #2785
	t.Run("put CUD with emptied photo", func(t *testing.T) {
		test := test()
		cud := ev1.CUDBuilder().Create(test.tablePhotos)
		cud.PutRecordID(appdef.SystemField_ID, test.tempPhotoID+1)
		cud.PutString(test.buyerIdent, test.buyerValue)
		cud.PutBytes(test.photoIdent, nil) // empty bytes-field
	})

	b := ev1.storeToBytes()
	length := len(b)

	t.Run("load/store from truncated bytes", func(t *testing.T) {
		for i := 0; i < length; i++ {
			damaged := b[0:i]

			ev2 := newEmptyTestEvent()
			err := ev2.loadFromBytes(damaged)
			require.Error(err, fmt.Sprintf("unexpected success load event from bytes truncated at %d", i))
		}
	})

	t.Run("load/store from damaged bytes",
		// - fail (Panic or Error) or
		// - success read wrong data
		func(t *testing.T) {
			stat := make(map[string]int)
			for i := 0; i < length; i++ {
				b[i] ^= 255
				ev2 := newEmptyTestEvent()

				func() {
					defer func() {
						if err := recover(); err != nil {
							log.Verbose("%d: panic at read event: %v", i, err)
							stat["Panics"]++
						}
					}()
					if err := ev2.loadFromBytes(b); err != nil {
						log.Verbose("%d: error at load: %v\n", i, err)
						stat["Errors"]++
						return
					}
					log.Verbose("%d: success load: %v\n", i)
					stat["Success"]++
				}()

				b[i] ^= 255
			}
			log.Verbose("len: %d, stat: %v\n", length, stat)
		})
}

func Test_LoadStoreErrEvent_Bytes(t *testing.T) {
	require := require.New(t)
	test := test()

	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(test.appName)
	require.NoError(err)

	eventName := [3]appdef.QName{
		appdef.NullQName,
		appdef.NewQName("unknown", "command-name"),
		appdef.NewQName("invalid q name", ""),
	}
	eventBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	for i := 0; i < len(eventName); i++ {
		t.Run("load/store bad command name error event", func(t *testing.T) {
			bld := app.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						EventBytes:        eventBytes,
						HandlingPartition: test.partition,
						PLogOffset:        test.plogOfs,
						Workspace:         test.workspace,
						WLogOffset:        test.wlogOfs,
						QName:             eventName[i],
						RegisteredAt:      test.registeredTime,
					},
					Device:   test.device,
					SyncedAt: test.syncTime,
				})
			rawEvent, buildErr := bld.BuildRawEvent()
			require.Error(buildErr)
			require.NotNil(rawEvent)

			ev1 := rawEvent.(*eventType)
			ev1.setBuildError(buildErr)
			require.False(ev1.valid())

			b := ev1.storeToBytes()

			ev2 := newEmptyTestEvent()
			err = ev2.loadFromBytes(b)
			require.NoError(err)

			require.Equal(istructs.QNameForError, ev2.QName())
			require.False(ev2.Error().ValidEvent())
			require.Contains(buildErr.Error(), ev2.Error().ErrStr())
			require.Equal(eventName[i], ev2.Error().QNameFromParams())
			require.Equal(eventBytes, ev2.Error().OriginalEventBytes())
		})
	}

	t.Run("load/store custom build error event", func(t *testing.T) {

		const (
			errMsg = "test build error message; エラーメッセージテスト; 🏐;"
			maxLen = 0xFFFF
		)
		for testNo := 0; testNo < 3; testNo++ {
			var msg string
			switch testNo {
			case 0:
				msg = ""
			case 1:
				msg = errMsg
			case 2:
				for len(msg) < maxLen {
					msg += errMsg
				}
			}
			t.Run("load/store custom build error event", func(t *testing.T) {
				ev1 := newTestEvent(100500, 500)
				ev1.argUnlObj.clear() // to prevent EventBytes obfuscate
				ev1.setBuildError(errors.New(msg))

				b := ev1.storeToBytes()

				ev2 := newEmptyTestEvent()
				err = ev2.loadFromBytes(b)
				require.NoError(err)

				require.Equal(istructs.QNameForError, ev2.QName())
				require.Equal(istructs.Offset(100500), ev2.pLogOffs)
				require.Equal(istructs.Offset(500), ev2.wLogOffs)
				require.False(ev2.Error().ValidEvent())
				require.True(strings.HasPrefix(msg, ev2.Error().ErrStr()))
				require.Equal(test.saleCmdName, ev2.Error().QNameFromParams())
				require.Equal(test.eventRawBytes, ev2.Error().OriginalEventBytes())
			})
		}
	})
}

func Test_LoadErrorEvent_DamagedBytes(t *testing.T) {
	const errMsg = "test build error message; エラーメッセージテスト"

	require := require.New(t)

	ev1 := newTestEvent(100500, 500)
	ev1.argUnlObj.clear() // to prevent EventBytes obfuscate
	ev1.setBuildError(errors.New(errMsg))

	b := ev1.storeToBytes()

	length := len(b)
	for i := 0; i < length; i++ {
		damaged := b[0:i]

		ev2 := newEmptyTestEvent()
		err := ev2.loadFromBytes(damaged)
		require.Error(err, fmt.Sprintf("unexpected success load event from bytes truncated at %d", i))
	}
}

func Test_LoadStoreNullEvent_Bytes(t *testing.T) {
	require := require.New(t)

	ev1 := newEmptyTestEvent()
	b := ev1.storeToBytes()

	ev2 := newEmptyTestEvent()
	err := ev2.loadFromBytes(b)
	require.NoError(err)

	require.Equal(appdef.NullQName, ev2.QName())
}

func Test_ObjectMask(t *testing.T) {
	require := require.New(t)
	test := test()

	value := newObject(test.AppCfg, test.saleCmdDocName, nil)
	fillTestObject(value)

	value.maskValues()

	require.Equal(maskString, value.AsString(test.buyerIdent))
	require.Equal(int32(0), value.AsInt32(test.ageIdent))
	require.Equal(float32(0), value.AsFloat32(test.heightIdent))
	require.False(value.AsBool(test.humanIdent))
	require.Equal([]byte(nil), value.AsBytes(test.photoIdent))

	var basket istructs.IObject
	for c := range value.Children(test.basketIdent) {
		basket = c
		break
	}
	require.NotNil(basket)

	var cnt int
	for c := range basket.Children(test.goodIdent) {
		require.Equal(maskString, c.AsString(test.nameIdent))
		require.Equal(int64(0), c.AsInt64(test.codeIdent))
		require.Equal(float64(0), c.AsFloat64(test.weightIdent))
		cnt++
	}

	require.Equal(test.goodCount, cnt)
}

func Test_objectType_FillFromJSON(t *testing.T) {
	require := require.New(t)
	test := test()

	tests := []struct {
		name  string
		data  string
		check func(istructs.IObject, error)
	}{
		{"should be ok to fill from empty JSON",
			`{}`,
			func(o istructs.IObject, err error) {
				require.NoError(err)
				require.Equal(test.testObj, o.QName())
				require.EqualValues(0, o.AsInt32("int32"))
			}},
		{"should be ok to fill from JSON with nil values even for unknown fields",
			`{"int32": null, "unknown": null}`,
			func(o istructs.IObject, err error) {
				require.NoError(err)
				require.Equal(test.testObj, o.QName())
				require.EqualValues(0, o.AsInt32("int32"))
			}},
		{"should be ok to fill fields from JSON",
			`{"int32": 1, "int64": 2, "float32": 3.3, "float64": 4.4, "bool": true, "string": "test", "bytes": "AQID"}`,
			func(o istructs.IObject, err error) {
				require.NoError(err)
				require.Equal(test.testObj, o.QName())
				require.EqualValues(1, o.AsInt32("int32"))
				require.EqualValues(2, o.AsInt64("int64"))
				require.EqualValues(float32(3.3), o.AsFloat32("float32"))
				require.EqualValues(4.4, o.AsFloat64("float64"))
				require.EqualValues(true, o.AsBool("bool"))
				require.EqualValues("test", o.AsString("string"))
				require.EqualValues([]byte{1, 2, 3}, o.AsBytes("bytes"))
			}},
		{"should be ok to fill children from JSON",
			`{"int32": 1, "child": [{"int64": 1}, {"int64": 2}, {"int64": 3}]}`,
			func(o istructs.IObject, err error) {
				require.NoError(err)
				require.Equal(test.testObj, o.QName())
				require.EqualValues(1, o.AsInt32("int32"))
				require.Equal(3, func() int {
					cnt := 0
					for c := range o.Children("child") {
						cnt++
						require.EqualValues(cnt, c.AsInt64("int64"))
					}
					return cnt
				}())
			}},
		{"should be ok to fill with nil values",
			`{"int32": null, "bool": null, "string": null, "bytes": null}`,
			func(o istructs.IObject, err error) {
				require.NoError(err)
				require.Equal(test.testObj, o.QName())
				require.Zero(o.AsInt32("int32"))
				require.Zero(o.AsBool("bool"))
				require.Zero(o.AsString("string"))
				require.Zero(o.AsBytes("bytes"))
			}},
		{"should be error if unknown field in JSON",
			`{"unknown": 1}`,
			func(o istructs.IObject, err error) {
				require.Error(err, require.Is(ErrNameNotFoundError), require.Has("unknown"))
			}},
		{"should be error if invalid data type in JSON field",
			`{"int32": "error"}`,
			func(o istructs.IObject, err error) {
				require.Error(err, require.Is(ErrWrongFieldTypeError), require.Has("int32"))
			}},
		{"should be error if unknown container in JSON",
			`{"unknown": [{"int32": 1}]}`,
			func(o istructs.IObject, err error) {
				require.Error(err, require.Is(ErrNameNotFoundError), require.Has("unknown"))
			}},
		{"should be error if invalid data type in JSON container",
			`{"child": ["a","b"]}`,
			func(o istructs.IObject, err error) {
				require.Error(err, ErrWrongTypeError,
					require.Has("invalid type «string»"),
					require.Has("child «child[0]»"))
			}},
		{"should be error if invalid data type in JSON container",
			`{"child": ["a","b"]}`,
			func(o istructs.IObject, err error) {
				require.Error(err, ErrWrongTypeError,
					require.Has("invalid type «string»"),
					require.Has("child «child[0]»"))
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := test.AppStructs.ObjectBuilder(test.testObj)
			require.NotNil(b)

			var j map[string]any
			d := json.NewDecoder(strings.NewReader(tt.data))
			d.UseNumber()
			require.NoError(d.Decode(&j))
			b.FillFromJSON(j)

			tt.check(b.Build())
		})
	}

	t.Run("should be error on provide a value of a wrong type", func(t *testing.T) {
		b := test.AppStructs.ObjectBuilder(test.testObj)
		require.NotNil(b)
		j := map[string]any{
			"int32": uint8(42),
		}
		b.FillFromJSON(j)

		_, err := b.Build()
		require.ErrorIs(err, ErrWrongTypeError)
	})
}
