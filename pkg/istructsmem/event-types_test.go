/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	log "github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/singletons"
)

func TestEventBuilder_Core(t *testing.T) {
	require := require.New(t)
	test := test()

	// gets AppStructProvider and AppStructs
	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.AppStructs(test.appName)
	require.NoError(err)

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

			basket := cmd.ElementBuilder(test.basketIdent)
			basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)
			for i := 0; i < test.goodCount; i++ {
				good := basket.ElementBuilder(test.goodIdent)
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
			require.NoError(buildErr, buildErr)
			require.NotNil(rawEvent)
		})
	})

	t.Run("II. Save raw event to PLog & WLog and save Docs and CUDs demo", func(t *testing.T) {
		// 1. save to PLog
		var nextID = istructs.FirstBaseRecordID
		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr,
			func(rawID istructs.RecordID, def appdef.IDef) (storageID istructs.RecordID, err error) {
				require.True(rawID.IsRaw())
				storageID = nextID
				switch rawID {
				case test.tempPhotoID:
					require.Equal(test.tablePhotos, def.QName())
					photoID = storageID
				case test.tempRemarkID:
					require.Equal(test.tablePhotoRems, def.QName())
					remarkID = storageID
				case test.tempSaleID:
					require.Equal(test.saleCmdDocName, def.QName())
					saleID = storageID
				case test.tempBasketID:
					require.Equal(appdef.NewQName(test.pkgName, test.basketIdent), def.QName())
					basketID = storageID
				case test.tempGoodsID[0]:
					require.Equal(appdef.NewQName(test.pkgName, test.goodIdent), def.QName())
					goodsID[0] = storageID
				case test.tempGoodsID[1]:
					require.Equal(appdef.NewQName(test.pkgName, test.goodIdent), def.QName())
					goodsID[1] = storageID
				}
				nextID++
				return storageID, nil
			},
		)
		require.NoError(saveErr, saveErr)
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
		wLogEvent, err := app.Events().PutWlog(pLogEvent)
		require.NoError(err)
		defer wLogEvent.Release()

		testDbEvent(t, wLogEvent)

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
		require.NotEqual(idP, istructs.NullRecordID)
		require.NotEqual(idR, istructs.NullRecordID)
	})

	t.Run("III. Read event from PLog & PLog and reads CUD demo", func(t *testing.T) {

		t.Run("must be ok to read PLog", func(t *testing.T) {
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

			t.Run("test single record reading", func(t *testing.T) {
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

			t.Run("test sequential reading", func(t *testing.T) {
				var event istructs.IPLogEvent
				err := app.Events().ReadPLog(context.Background(), test.partition, test.plogOfs, istructs.ReadToTheEnd,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						require.Equal(test.plogOfs, plogOffset)
						event = ev
						return nil
					})
				defer event.Release() // not necessary
				check(event, err)
			})
		})

		t.Run("must be no events if read other PLog partition", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var event istructs.IPLogEvent
				err := app.Events().ReadPLog(context.Background(), test.partition+1, test.plogOfs, 1,
					func(plogOffset istructs.Offset, ev istructs.IPLogEvent) (err error) {
						require.Fail("must be no events if read other PLog partition")
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
						require.Fail("must be no events if read other PLog partition")
						event = ev
						return nil
					})
				require.NoError(err)
				require.Nil(event)
			})
		})

		t.Run("must be ok to read WLog", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var event istructs.IWLogEvent
				err = app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs, 1,
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
				err = app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs, 1,
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

		t.Run("must be no event if read WLog from other WSID", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var wLogEvent istructs.IWLogEvent
				err = app.Events().ReadWLog(context.Background(), test.workspace+1, test.wlogOfs, 1,
					func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
						require.Fail("must be no event if read WLog from other WSID")
						return nil
					})
				require.NoError(err)
				require.Nil(wLogEvent)
			})

			t.Run("test sequential reading", func(t *testing.T) {
				var wLogEvent istructs.IWLogEvent
				err = app.Events().ReadWLog(context.Background(), test.workspace+1, test.wlogOfs, istructs.ReadToTheEnd,
					func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
						require.Fail("must be no event if read WLog from other WSID")
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
		changedRems    string  = "changes изменение"
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
			require.NoError(buildErr, buildErr)
			require.NotNil(rawEvent)
		})
	})

	t.Run("VII. Save change event to PLog & WLog and save CUD demo", func(t *testing.T) {

		var pLogEvent istructs.IPLogEvent

		t.Run("test save to PLog", func(t *testing.T) {
			ev, saveErr := app.Events().PutPlog(rawEvent, buildErr,
				func(_ istructs.RecordID, _ appdef.IDef) (storageID istructs.RecordID, err error) {
					return 0, nil // no new records
				},
			)
			require.NoError(saveErr, saveErr)
			require.NotNil(ev)
			pLogEvent = ev
		})
		defer pLogEvent.Release()

		t.Run("test save to WLog", func(t *testing.T) {
			wLogEvent, err := app.Events().PutWlog(pLogEvent)
			require.NoError(err)
			require.NotNil(wLogEvent)
			defer wLogEvent.Release()
		})

		t.Run("test apply PLog event records", func(t *testing.T) {
			err = app.Records().Apply(pLogEvent)
			require.NoError(err)
		})
	})

	t.Run("VIII. Read event from PLog & WLog and reads CUD", func(t *testing.T) {

		checkEvent := func(event istructs.IDbEvent, err error) {
			require.NoError(err)
			require.NotNil(event)

			t.Run("test PLog event CUDs", func(t *testing.T) {
				cudCount := 0
				event.CUDs(func(rec istructs.ICUDRow) error {
					if rec.QName() == test.tablePhotos {
						require.False(rec.IsNew())
						require.Equal(changedHeights, rec.AsFloat32(test.heightIdent))
						require.Equal(changedPhoto, rec.AsBytes(test.photoIdent))
					}
					if rec.QName() == test.tablePhotoRems {
						require.False(rec.IsNew())
						require.Equal(changedRems, rec.AsString(test.remarkIdent))
					}
					cudCount++
					return nil
				})
				require.Equal(2, cudCount)

				t.Run("test event CUDs (update) breakable by error", func(t *testing.T) {
					testError := errors.New("test error")
					err := event.CUDs(func(rec istructs.ICUDRow) error {
						return testError
					})
					require.ErrorIs(err, testError)
				})
			})
		}

		t.Run("must be ok to read PLog", func(t *testing.T) {

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

		t.Run("must be ok to read WLog", func(t *testing.T) {

			t.Run("test single record reading", func(t *testing.T) {
				var event istructs.IWLogEvent
				err = app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs+1, 1,
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
				err = app.Events().ReadWLog(context.Background(), test.workspace, test.wlogOfs+1, istructs.ReadToTheEnd,
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
			bytes, err := r.storeToBytes()
			require.NoError(err)
			require.True(len(bytes) > 0)
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
			err = app.Events().ReadPLog(context.Background(), test.partition, test.plogOfs+1, 1,
				func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
					require.Equal(test.plogOfs+1, plogOffset)
					pLogEvent = event
					return nil
				})
			require.NoError(err)
			require.NotNil(pLogEvent)
			defer pLogEvent.Release()

			checked := false
			pLogEvent.CUDs(func(rec istructs.ICUDRow) error {
				if rec.QName() == test.tablePhotos {
					require.False(rec.IsNew())
					require.Equal(changedHeights, rec.AsFloat32(test.heightIdent))
					require.Equal(changedPhoto, rec.AsBytes(test.photoIdent))
					checked = true
				}
				return nil
			})

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
						require.FailNow("unexpected record QName from Apply2 to callback returned: «%v»", r.QName())
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

	var basket istructs.IElement

	t.Run("test basket", func(t *testing.T) {
		var names []string
		cmd.Containers(
			func(name string) { names = append(names, name) })
		require.Equal(1, len(names))
		require.Equal(test.basketIdent, names[0])

		cmd.Elements(test.basketIdent, func(nest istructs.IElement) { basket = nest })
		require.NotNil(basket)

		require.Equal(cmd.AsRecord().ID(), basket.AsRecord().Parent())
	})

	t.Run("test goods", func(t *testing.T) {
		var names []string
		basket.Containers(
			func(name string) { names = append(names, name) })
		require.Equal(len(names), 1)
		require.Equal(test.goodIdent, names[0])

		var goods []istructs.IElement
		basket.Elements(test.goodIdent, func(good istructs.IElement) { goods = append(goods, good) })
		require.NotNil(goods)
		require.Equal(test.goodCount, len(goods))

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
	cmd.FieldNames(func(fieldName string) {
		if fieldName == test.passwordIdent {
			hasPassword = true
		}
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
		err := event.CUDs(func(row istructs.ICUDRow) error {
			cuds = append(cuds, row)
			if cnt == 0 {
				require.True(row.IsNew())
				require.Equal(test.tablePhotos, row.QName())
			}
			cnt++
			return nil
		})
		require.NoError(err)
		require.Equal(2, cnt)
		require.Equal(2, len(cuds))
		testPhotoRow(t, cuds[0])
		require.Equal(cuds[0].AsRecordID(appdef.SystemField_ID), cuds[1].AsRecordID(test.photoIdent))
		require.Equal(test.remarkValue, cuds[1].AsString(test.remarkIdent))

		t.Run("test event CUDs (create) breakable by error", func(t *testing.T) {
			testErr := errors.New("test error")
			err := event.CUDs(func(rec istructs.ICUDRow) error {
				return testErr
			})
			require.ErrorIs(err, testErr)
		})
	})
}

func Test_EventUpdateRawCud(t *testing.T) {
	// this test for https://dev.heeus.io/launchpad/#!25853
	require := require.New(t)

	docName := appdef.NewQName("test", "cDoc")
	recName := appdef.NewQName("test", "cRec")

	appDef := appdef.New()

	t.Run("must ok to construct application definition", func(t *testing.T) {
		doc := appDef.AddCDoc(docName)
		doc.AddField("new", appdef.DataKind_bool, true)
		doc.AddField("rec", appdef.DataKind_RecordID, false)
		doc.AddField("emptied", appdef.DataKind_string, false)
		doc.AddContainer("rec", recName, 0, 1)

		rec := appDef.AddCRecord(recName)
		rec.AddField("data", appdef.DataKind_string, false)
	})

	cfgs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)
		cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		return cfgs
	}()

	const (
		simpleTest uint = iota
		retryTest

		testCount
	)

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	ws := istructs.WSID(1)

	for test := simpleTest; test < testCount; test++ {

		app, err := provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)

		docID := istructs.RecordID(322685000131087 + test)

		t.Run("must ok to create CDoc", func(t *testing.T) {
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

			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err,
				func(rawID istructs.RecordID, def appdef.IDef) (storageID istructs.RecordID, err error) {
					require.EqualValues(1, rawID)
					require.EqualValues(docName, def.QName())
					return docID, nil
				})
			require.NotNil(pLogEvent)
			require.NoError(saveErr, saveErr)
			require.True(pLogEvent.Error().ValidEvent())

			t.Run("must ok to apply CDoc records", func(t *testing.T) {
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

		recID := istructs.RecordID(322685000131087 + test + 1)

		t.Run("must ok to update CDoc", func(t *testing.T) {
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

			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err,
				func(rawID istructs.RecordID, def appdef.IDef) (storageID istructs.RecordID, err error) {
					require.EqualValues(1, rawID)
					require.EqualValues(recName, def.QName())
					return recID, nil
				})
			require.NotNil(pLogEvent)
			require.NoError(saveErr, saveErr)
			require.True(pLogEvent.Error().ValidEvent())

			switch test {
			case retryTest:
				t.Run("must ok to reread PLog event", func(t *testing.T) {
					pLogEvent = nil
					err := app.Events().ReadPLog(context.Background(), 1, istructs.Offset(100501+test), 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
						require.EqualValues(100501+test, plogOffset)
						pLogEvent = event
						return nil
					})
					require.NoError(err)
					require.NotNil(pLogEvent)
					defer pLogEvent.Release()
					require.True(pLogEvent.Error().ValidEvent())
				})
			}

			t.Run("must ok to apply CDoc records", func(t *testing.T) {
				recCnt := 0
				err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
					switch id := r.ID(); id {
					case docID:
						require.EqualValues(docName, r.QName())
						require.EqualValues(r.AsRecordID("rec"), recID, "error #25853 here!")
					case recID:
						require.EqualValues(recName, r.QName())
						require.EqualValues(r.Parent(), docID)
						require.EqualValues(r.Container(), "rec")
						require.EqualValues(r.AsString("data"), "test data")
					default:
						require.Fail("unexpected record applied")
					}
					recCnt++
				})
				require.Equal(2, recCnt)
			})

			t.Run("must ok to reread CDoc record", func(t *testing.T) {
				rec, err := app.Records().Get(ws, true, docID)
				require.NoError(err)
				require.EqualValues(docName, rec.QName())
				require.EqualValues(rec.AsRecordID("rec"), recID, "error #25853 here!")
			})
		})

	}
}

func Test_SingletonCDocEvent(t *testing.T) {
	require := require.New(t)

	docName := appdef.NewQName("test", "cDoc")
	docID := istructs.NullRecordID

	appDef := appdef.New()

	t.Run("must ok to construct singleton CDoc", func(t *testing.T) {
		def := appDef.AddSingleton(docName)
		def.AddField("option", appdef.DataKind_int64, true)
	})

	cfgs := func() AppConfigsType {
		cfgs := make(AppConfigsType, 1)
		cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		return cfgs
	}()

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)

	docID, err = cfgs.GetConfig(istructs.AppQName_test1_app1).singletons.ID(docName)
	require.NoError(err)

	t.Run("must ok to read not created singleton CDoc by QName", func(t *testing.T) {
		rec, err := app.Records().GetSingleton(1, docName)
		require.NoError(err)
		require.Equal(appdef.NullQName, rec.QName())
		require.Equal(docID, rec.ID())
	})

	t.Run("must ok to create singleton CDoc", func(t *testing.T) {
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

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err,
			func(_ istructs.RecordID, _ appdef.IDef) (storageID istructs.RecordID, err error) {
				return istructs.NullRecordID, fmt.Errorf("unexpected call ID generator from singleton CDoc creation")
			})
		require.NotNil(pLogEvent)
		require.NoError(saveErr, saveErr)
		require.True(pLogEvent.Error().ValidEvent())

		t.Run("newly created singleton CDoc must be ok", func(t *testing.T) {
			recCnt := 0
			pLogEvent.CUDs(func(rec istructs.ICUDRow) error {
				require.Equal(docName, rec.QName())
				require.Equal(docID, rec.ID())
				require.True(rec.IsNew())
				require.Equal(int64(8), rec.AsInt64("option"))
				recCnt++
				return nil
			})
			require.Equal(1, recCnt)
		})

		t.Run("must ok to apply singleton CDoc records", func(t *testing.T) {
			recCnt := 0
			err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
				require.Equal(docName, r.QName())
				require.Equal(docID, r.ID())
				require.Equal(int64(8), r.AsInt64("option"))
				recCnt++
			})
			require.Equal(1, recCnt)
		})

		t.Run("must fail if attempt to reapply singleton CDoc creation", func(t *testing.T) {
			err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
				require.Fail("must fail if attempt to reapply singleton CDoc creation")
			})
			require.ErrorIs(err, ErrRecordIDUniqueViolation)
		})
	})

	t.Run("must ok to read singleton CDoc by QName", func(t *testing.T) {
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
		require.ErrorIs(buildErr, ErrRecordIDUniqueViolation)

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr,
			func(_ istructs.RecordID, _ appdef.IDef) (storageID istructs.RecordID, err error) {
				return istructs.NullRecordID, fmt.Errorf("unexpected call ID generator from singleton CDoc creation")
			})
		require.NotNil(pLogEvent)
		require.NoError(saveErr, saveErr)
		require.False(pLogEvent.Error().ValidEvent())

		require.Panics(
			func() {
				_ = app.Records().Apply2(pLogEvent, func(_ istructs.IRecord) {})
			},
			"must panic if apply invalid event")
	})

	t.Run("must ok to update singleton CDoc", func(t *testing.T) {
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

		pLogEvent, saveErr := app.Events().PutPlog(rawEvent, err,
			func(_ istructs.RecordID, _ appdef.IDef) (storageID istructs.RecordID, err error) {
				return istructs.NullRecordID, fmt.Errorf("unexpected call ID generator while singleton CDoc update")
			})
		require.NotNil(pLogEvent)
		require.NoError(saveErr, saveErr)
		require.True(pLogEvent.Error().ValidEvent())

		t.Run("updated singleton CDoc must be ok", func(t *testing.T) {
			recCnt := 0
			pLogEvent.CUDs(func(rec istructs.ICUDRow) error {
				require.Equal(docName, rec.QName())
				require.Equal(docID, rec.ID())
				require.False(rec.IsNew())
				require.Equal(int64(888), rec.AsInt64("option"))
				recCnt++
				return nil
			})
			require.Equal(1, recCnt)
		})

		t.Run("must ok to apply singleton CDoc update records", func(t *testing.T) {
			recCnt := 0
			err = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {
				require.Equal(docName, r.QName())
				require.Equal(docID, r.ID())
				require.Equal(int64(888), r.AsInt64("option"))
				recCnt++
			})
			require.Equal(1, recCnt)
		})

		t.Run("must ok to read updated singleton CDoc from records", func(t *testing.T) {
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

	app, err := provider.AppStructs(test.appName)
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
		require.ErrorIs(buildErr, ErrCUDsMissed)
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

	t.Run("Error in ArgumentObject inner element", func(t *testing.T) {
		bld := eventBuilder(test.saleCmdName)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		basket := cmd.ElementBuilder(test.basketIdent)
		basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)
		good := basket.ElementBuilder(test.goodIdent)
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
		basket := cmd.ElementBuilder(test.basketIdent)
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
		basket := cmd.ElementBuilder(test.basketIdent)
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
			return &r
		}

		t.Run("update not applicable by QName", func(t *testing.T) {
			bld := eventBuilder(test.changeCmdName)

			rec := getPhoto()

			cud := bld.CUDBuilder().Update(rec)
			cud.PutQName(appdef.SystemField_QName, test.tablePhotoRems)
			cud.PutRecordID(appdef.SystemField_ID, 100501)
			cud.PutRecordID(appdef.SystemField_ParentID, 100500)
			cud.PutString(appdef.SystemField_Container, test.remarkIdent)
			cud.PutRecordID(test.photoIdent, 100500)
			cud.PutString(test.remarkIdent, test.remarkValue)

			_, buildErr = bld.BuildRawEvent()
			require.ErrorIs(buildErr, ErrDefChanged)
		})

		t.Run("update unknown field", func(t *testing.T) {
			bld := eventBuilder(test.changeCmdName)

			rec := getPhoto()

			cud := bld.CUDBuilder().Update(rec)
			cud.PutFloat32("unknownField", 7.7)

			_, buildErr = bld.BuildRawEvent()
			require.ErrorIs(buildErr, ErrNameNotFound)
		})

		t.Run("can`t change system fields", func(t *testing.T) {

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
				return &r
			}

			t.Run("can`t change sys.ID", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutRecordID(appdef.SystemField_ID, 100502)
				_, buildErr = bld.BuildRawEvent()
				require.ErrorIs(buildErr, ErrUnableToUpdateSystemField)
			})

			t.Run("can`t change sys.ParentID", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutRecordID(appdef.SystemField_ParentID, 100502)
				_, buildErr = bld.BuildRawEvent()
				require.ErrorIs(buildErr, ErrUnableToUpdateSystemField)
			})

			t.Run("can`t change sys.Container", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutString(appdef.SystemField_Container, test.basketIdent) // error here
				_, buildErr = bld.BuildRawEvent()
				require.ErrorIs(buildErr, ErrUnableToUpdateSystemField)
			})

			t.Run("allow to change sys.IsActive", func(t *testing.T) {
				bld := eventBuilder(test.changeCmdName)
				cud := bld.CUDBuilder().Update(getPhotoRem())
				cud.PutBool(appdef.SystemField_IsActive, false)
				rawEvent, buildErr = bld.BuildRawEvent()
				require.NoError(buildErr, buildErr)
				require.NotNil(rawEvent)
			})
		})

		t.Run("update has unknown field", func(t *testing.T) {
			bld := eventBuilder(test.changeCmdName)

			rec := getPhoto()

			cud := bld.CUDBuilder().Update(rec)
			cud.PutString("unknown field", "someValue")

			rawEvent, buildErr = bld.BuildRawEvent()
			require.ErrorIs(buildErr, ErrNameNotFound)
			require.NotNil(rawEvent)
		})

	})

	t.Run("Errors in Generate ID", func(t *testing.T) {

		t.Run("Error in Generate Argument Object ID", func(t *testing.T) {
			bld := eventBuilder(test.saleCmdName)

			cmd := bld.ArgumentObjectBuilder()
			cmd.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
			cmd.PutString(test.buyerIdent, test.buyerValue)
			basket := cmd.ElementBuilder(test.basketIdent)
			basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)

			cmdSec := bld.ArgumentUnloggedObjectBuilder()
			cmdSec.PutString(test.passwordIdent, "12345")

			rawEvent, buildErr = bld.BuildRawEvent()
			require.NoError(buildErr, buildErr)
			require.NotNil(rawEvent)

			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr,
				func(tempId istructs.RecordID, def appdef.IDef) (storageID istructs.RecordID, err error) {
					if tempId == test.tempBasketID {
						require.Equal(appdef.NewQName(test.pkgName, test.basketIdent), def.QName())
						return istructs.NullRecordID, fmt.Errorf("test error: %w", ErrWrongRecordID)
					}
					return 100500, nil
				})
			require.False(pLogEvent.Error().ValidEvent())
			require.Contains(pLogEvent.Error().ErrStr(), ErrWrongRecordID.Error())
			require.NoError(saveErr, saveErr)
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
			require.NoError(buildErr, buildErr)
			require.NotNil(rawEvent)

			pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr,
				func(tempId istructs.RecordID, def appdef.IDef) (storageID istructs.RecordID, err error) {
					if tempId == 7 {
						require.Equal(test.tablePhotoRems, def.QName())
						return istructs.NullRecordID, fmt.Errorf("test error: %w", ErrWrongRecordID)
					}
					return 100500, nil
				})
			require.False(pLogEvent.Error().ValidEvent())
			require.Contains(pLogEvent.Error().ErrStr(), ErrWrongRecordID.Error())
			require.NoError(saveErr, saveErr)
			require.NotNil(pLogEvent)

			require.Panics(
				func() {
					_ = app.Records().Apply2(pLogEvent, func(r istructs.IRecord) {})
				},
				"must panic if apply invalid event")
		})
	})
}

func Test_LoadStoreEvent_Bytes(t *testing.T) {
	require := require.New(t)

	ev1 := newTestEvent(100500, 500)
	testDbEvent(t, ev1)

	ev1.argUnlObj.maskValues()

	b, err := ev1.storeToBytes()
	require.NoError(err)

	ev2 := newEmptyTestEvent()
	err = ev2.loadFromBytes(b)
	require.NoError(err)

	require.Equal(istructs.Offset(100500), ev2.pLogOffs)
	require.Equal(istructs.Offset(500), ev2.wLogOffs)

	testDbEvent(t, ev2)
	testUnloggedObject(t, ev2.ArgumentUnloggedObject())
}

func Test_LoadEvent_CorruptedBytes(t *testing.T) {
	require := require.New(t)

	ev1 := newTestEvent(100500, 500)
	testDbEvent(t, ev1)

	b, err := ev1.storeToBytes()
	require.NoError(err)

	len := len(b)

	t.Run("load/store from truncated bytes", func(t *testing.T) {
		for i := 0; i < len; i++ {
			corrupted := b[0:i]

			ev2 := newEmptyTestEvent()
			err = ev2.loadFromBytes(corrupted)
			require.Error(err, fmt.Sprintf("unexpected success load event from bytes truncated at %d", i))
		}
	})

	t.Run("load/store from corrupted bytes:\n"+
		"— fail (Panic or Error) or\n"+
		"— success read wrong data",
		func(t *testing.T) {
			stat := make(map[string]int)
			for i := 0; i < len; i++ {
				b[i] ^= 255
				ev2 := newEmptyTestEvent()

				func() {
					defer func() {
						if err := recover(); err != nil {
							log.Verbose("%d: panic at read event: %v", i, err)
							stat["Panics"]++
						}
					}()
					if err = ev2.loadFromBytes(b); err != nil {
						log.Verbose("%d: error at load: %v\n", i, err)
						stat["Errors"]++
						return
					}
					log.Verbose("%d: success load: %v\n", i)
					stat["Success"]++
				}()

				b[i] ^= 255
			}
			log.Verbose("len: %d, stat: %v\n", len, stat)
		})
}

func Test_LoadStoreErrEvent_Bytes(t *testing.T) {
	require := require.New(t)
	test := test()

	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.AppStructs(test.appName)
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

			ev1 := newEmptyTestEvent()
			ev1.eventType.copyFrom(rawEvent.(*eventType))
			ev1.setBuildError(buildErr)
			require.False(ev1.valid())

			b, err := ev1.storeToBytes()
			require.NoError(err)

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
			errMsg = "test build error message; тест сообщения об ошибке; エラーメッセージテスト; 🏐;"
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

				b, err := ev1.storeToBytes()
				require.NoError(err)

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

func Test_LoadErrorEvent_CorruptedBytes(t *testing.T) {
	const errMsg = "test build error message; тест сообщения об ошибке; エラーメッセージテスト"

	require := require.New(t)

	ev1 := newTestEvent(100500, 500)
	ev1.argUnlObj.clear() // to prevent EventBytes obfuscate
	ev1.setBuildError(errors.New(errMsg))

	b, err := ev1.storeToBytes()
	require.NoError(err)

	len := len(b)
	for i := 0; i < len; i++ {
		corrupted := b[0:i]

		ev2 := newEmptyTestEvent()
		err = ev2.loadFromBytes(corrupted)
		require.Error(err, fmt.Sprintf("unexpected success load event from bytes truncated at %d", i))
	}
}

func Test_LoadStoreNullEvent_Bytes(t *testing.T) {
	require := require.New(t)

	ev1 := newEmptyTestEvent()
	b, err := ev1.storeToBytes()
	require.NoError(err)

	ev2 := newEmptyTestEvent()
	err = ev2.loadFromBytes(b)
	require.NoError(err)

	require.Equal(appdef.NullQName, ev2.QName())
}

func Test_ObjectMask(t *testing.T) {
	require := require.New(t)
	test := test()

	value := newObject(test.AppCfg, test.saleCmdDocName)
	fillTestObject(&value)

	value.maskValues()

	require.Equal(maskString, value.AsString(test.buyerIdent))
	require.Equal(int32(0), value.AsInt32(test.ageIdent))
	require.Equal(float32(0), value.AsFloat32(test.heightIdent))
	require.False(value.AsBool(test.humanIdent))
	require.Equal([]byte(nil), value.AsBytes(test.photoIdent))

	var basket istructs.IElement
	value.Elements(test.basketIdent, func(el istructs.IElement) { basket = el })
	require.NotNil(basket)

	var cnt int
	basket.Elements(test.goodIdent, func(el istructs.IElement) {
		require.Equal(maskString, el.AsString(test.nameIdent))
		require.Equal(int64(0), el.AsInt64(test.codeIdent))
		require.Equal(float64(0), el.AsFloat64(test.weightIdent))
		cnt++
	})

	require.Equal(test.goodCount, cnt)
}
