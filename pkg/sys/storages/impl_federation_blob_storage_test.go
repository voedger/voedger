/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestFederationBlobStorage_BasicUsage(t *testing.T) {
	buffer := make([]byte, 5000)
	buffer[0] = 1
	buffer[4999] = 2
	require := require.New(t)
	ownerRecordExpected := appdef.NewQName("test", "blobOwner")
	federatioBlobHandler := func(owner, appname string, wsid istructs.WSID, ownerRecord appdef.QName, ownerRecordField appdef.FieldName,
		ownerID istructs.RecordID) (result []byte, err error) {
		require.Equal("owner", owner)
		require.Equal("appname", appname)
		require.Equal(istructs.WSID(123), wsid)
		require.Equal(istructs.RecordID(1), ownerID)
		require.Equal("recordField", ownerRecordField)
		require.Equal(ownerRecordExpected, ownerRecord)
		return buffer, nil
	}
	mockedStructs, _ := mockedStructs(t)
	appStructsFunc := func() istructs.IAppStructs {
		return mockedStructs
	}
	storage := NewFederationBlobStorage(appStructsFunc, nil, nil, nil, federatioBlobHandler)
	k := storage.NewKeyBuilder(appdef.NullQName, nil)
	k.PutString(sys.Storage_FederationBlob_Field_Owner, "owner")
	k.PutString(sys.Storage_FederationBlob_Field_AppName, "appname")
	k.PutRecordID(sys.Storage_FederationBlob_Field_OwnerID, 1)
	k.PutString(sys.Storage_FederationBlob_Field_OwnerRecordField, "recordField")
	k.PutQName(sys.Storage_FederationBlob_Field_OwnerRecord, ownerRecordExpected)
	k.PutString(sys.Storage_FederationBlob_Field_ExpectedCodes, "200,201")
	k.PutInt64(sys.Storage_FederationBlob_Field_WSID, 123)

	readBytes := make([]byte, 0)
	err := storage.(state.IWithRead).Read(k, func(_ istructs.IKey, value istructs.IStateValue) (err error) {
		readBytes = append(readBytes, value.AsBytes(sys.Storage_FederationBlob_Field_Body)...)
		return
	})
	require.NoError(err)
	require.Len(readBytes, 5000)
	require.Equal(byte(1), readBytes[0])
	require.Equal(byte(2), readBytes[4999])
}
