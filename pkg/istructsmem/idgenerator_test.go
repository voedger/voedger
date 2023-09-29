/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

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

			idGen.UpdateOnSync(storageID, appDef.Type(qName))
			storageIDNew, err := idGen.NextID(1, appDef.Type(qName))
			require.NoError(err)
			require.Equal(storageID+1, storageIDNew)

			idGen.UpdateOnSync(storageIDNew+100, appDef.Type(qName))
			storageIDNew, err = idGen.NextID(1, appDef.Type(qName))
			require.NoError(err)
			require.Equal(storageID+102, storageIDNew)

			idGen.UpdateOnSync(storageIDNew-1, appDef.Type(qName))
			storageIDNew, err = idGen.NextID(1, appDef.Type(qName))
			require.NoError(err)
			require.Equal(storageID+103, storageIDNew)
		}
	})
}

