/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package istructsmem

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestID(t *testing.T) {
	recID := istructs.RecordID(5000000000000)
	log.Println(recID.BaseRecordID())
	log.Println(istructs.NewCDocCRecordID(0))
	log.Println(istructs.NewCDocCRecordID(1))
	log.Println(istructs.NewCDocCRecordID(istructs.FirstBaseRecordID))
	recID = istructs.RecordID(5000000000333)
	log.Println(recID.BaseRecordID())
	log.Println(istructs.NewCDocCRecordID(recID.BaseRecordID()))
	recID = istructs.RecordID(10000008911)
	log.Println(recID.BaseRecordID())
	log.Println(istructs.NewCDocCRecordID(recID.BaseRecordID()))
	recID = istructs.RecordID(9999999996)
	log.Println(recID.BaseRecordID())
	log.Println(istructs.NewCDocCRecordID(recID.BaseRecordID()))
	recID = istructs.RecordID(5000000333)
	log.Println(recID.BaseRecordID())
	log.Println(istructs.NewCDocCRecordID(recID.BaseRecordID()))
	log.Println(istructs.NewRecordID(istructs.NullRecordID))
}

func TestIDGenerator(t *testing.T) {
	require := require.New(t)
	bld := appdef.New()
	qNameCDoc := appdef.NewQName(appdef.SysPackage, "cdoc")
	qNameCRecord := appdef.NewQName(appdef.SysPackage, "crecord")
	qNameWDoc := appdef.NewQName(appdef.SysPackage, "wdoc")
	bld.AddCDoc(qNameCDoc)
	bld.AddCRecord(qNameCRecord)
	bld.AddWDoc(qNameWDoc)
	appDef, err := bld.Build()
	require.NoError(err)

	idGen := NewIDGenerator()
	t.Run("basic usage", func(t *testing.T) {

		expectedCRecordID := istructs.NewCDocCRecordID(istructs.FirstBaseRecordID)

		storageID, err := idGen.NextID(1, appDef.Type(qNameCDoc))
		require.NoError(err)
		require.Equal(expectedCRecordID, storageID)

		expectedCRecordID++
		storageID, err = idGen.NextID(1, appDef.Type(qNameCDoc))
		require.NoError(err)
		require.Equal(expectedCRecordID, storageID)

		expectedCRecordID++
		storageID, err = idGen.NextID(1, appDef.Type(qNameCRecord))
		require.NoError(err)
		require.Equal(expectedCRecordID, storageID)

		expectedRecordID := istructs.NewRecordID(istructs.FirstBaseRecordID)
		storageID, err = idGen.NextID(1, appDef.Type(qNameWDoc))
		require.NoError(err)
		require.Equal(expectedRecordID, storageID)

		expectedRecordID++
		storageID, err = idGen.NextID(1, appDef.Type(qNameWDoc))
		require.NoError(err)
		require.Equal(expectedRecordID, storageID)
	})

	t.Run("UpdateOnSync", func(t *testing.T) {
		qNames := []appdef.QName{qNameCDoc, qNameWDoc}
		for _, qName := range qNames {
			storageID, err := idGen.NextID(1, appDef.Type(qName))
			require.NoError(err)

			idGen.UpdateOnSync(storageID+1, appDef.Type(qName))
			storageIDNew, err := idGen.NextID(1, appDef.Type(qName))
			require.NoError(err)
			require.Equal(storageID+2, storageIDNew)

			idGen.UpdateOnSync(storageIDNew+100, appDef.Type(qName))
			storageIDNew, err = idGen.NextID(1, appDef.Type(qName))
			require.NoError(err)
			require.Equal(storageID+103, storageIDNew)

			require.Panics(func() { idGen.UpdateOnSync(storageIDNew-1, appDef.Type(qName)) })
		}
	})
}

// https://github.com/voedger/voedger/issues/688
// 9999999999 ID causes next IDs collision
func TestIDGenCollision(t *testing.T) {
	t.Skip("fixed already. The test is kept as the problem description. Will work on commit e.g. https://github.com/voedger/voedger/commit/cbf1fec92fe1ec25fa17b9897261835c7aa6c017")
	require := require.New(t)
	idGen := NewIDGenerator()
	qNameCDoc := appdef.NewQName(appdef.SysPackage, "cdoc")
	bld := appdef.New()
	bld.AddCDoc(qNameCDoc)
	appDef, err := bld.Build()
	require.NoError(err)
	tp := appDef.Type(qNameCDoc)

	// server starts, 9999999999 record is met
	storedRecID := istructs.RecordID(9999999999)
	idGen.UpdateOnSync(storedRecID, tp)

	// let's work, query the next ID
	newIDBeforeRestart, err := idGen.NextID(1, tp)
	require.NoError(err)
	log.Println("ID generated before restart:", newIDBeforeRestart)
	// assume id 322690000000000 is inserted

	// server restarts 9B record is met again on recovery
	idGen = NewIDGenerator()
	storedRecID = istructs.RecordID(9999999999)
	idGen.UpdateOnSync(storedRecID, tp)
	// then record 322690000000000 is met
	storedRecID = istructs.RecordID(322690000000000)
	log.Println("322690000000000.BaseID = ", storedRecID.BaseRecordID())
	idGen.UpdateOnSync(storedRecID, tp) // its BaseRecordID is 0 which is <idGen.nextCDocCRecordBaseID(5000000000) so idGen.nextCDocCRecordBaseID is still 5000000000
	require.NoError(err)

	// now let's work, query the next ID
	// but it is still 322690000000000 because nextCDocCRecordBaseID was not bumped on UpdateOnSync()
	newIDAfterRestart, err := idGen.NextID(1, tp)
	require.NoError(err)
	log.Println("ID generated after restart:", newIDAfterRestart)
	require.Equal(newIDBeforeRestart, newIDAfterRestart) // should not be equal. 9999999999 ID causes next IDs collision
}
