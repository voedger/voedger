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
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestIDGenerator(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	wsb.AddCDoc(istructs.QNameCDoc)
	wsb.AddCRecord(istructs.QNameCRecord)
	wsb.AddWDoc(istructs.QNameWDoc)
	appDef, err := adb.Build()
	require.NoError(err)

	idGen := NewIDGenerator()
	t.Run("basic usage", func(t *testing.T) {

		expectedCRecordID := istructs.NewCDocCRecordID(istructs.FirstBaseRecordID)

		storageID, err := idGen.NextID(1, appDef.Type(istructs.QNameCDoc))
		require.NoError(err)
		require.Equal(expectedCRecordID, storageID)

		expectedCRecordID++
		storageID, err = idGen.NextID(1, appDef.Type(istructs.QNameCDoc))
		require.NoError(err)
		require.Equal(expectedCRecordID, storageID)

		expectedCRecordID++
		storageID, err = idGen.NextID(1, appDef.Type(istructs.QNameCRecord))
		require.NoError(err)
		require.Equal(expectedCRecordID, storageID)

		expectedRecordID := istructs.NewRecordID(istructs.FirstBaseRecordID)
		storageID, err = idGen.NextID(1, appDef.Type(istructs.QNameWDoc))
		require.NoError(err)
		require.Equal(expectedRecordID, storageID)

		expectedRecordID++
		storageID, err = idGen.NextID(1, appDef.Type(istructs.QNameWDoc))
		require.NoError(err)
		require.Equal(expectedRecordID, storageID)
	})

	t.Run("UpdateOnSync", func(t *testing.T) {
		qNames := []appdef.QName{istructs.QNameCDoc, istructs.QNameWDoc}
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
		}
	})
}

// https://github.com/voedger/voedger/issues/688
// 9999999999 ID causes next IDs collision
func TestIDGenCollision(t *testing.T) {
	t.Skip("fixed already. The test is kept as the problem description. The test is actual for commit e.g. https://github.com/voedger/voedger/commit/cbf1fec92fe1ec25fa17b9897261835c7aa6c017")

	require := require.New(t)

	idGen := NewIDGenerator()
	adb := builder.New()
	ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
	ws.AddCDoc(istructs.QNameCDoc)
	appDef, err := adb.Build()
	require.NoError(err)
	tp := appDef.Type(istructs.QNameCDoc)

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
