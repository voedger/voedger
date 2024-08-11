/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func TestFederationBlobStorage_BasicUsage(t *testing.T) {
	buffer := make([]byte, 5000)
	buffer[0] = 1
	buffer[4999] = 2
	require := require.New(t)
	federatioBlobHandler := func(owner, appname string, wsid istructs.WSID, blobId int64) (result []byte, err error) {
		require.Equal("owner", owner)
		require.Equal("appname", appname)
		require.Equal(istructs.WSID(123), wsid)
		require.Equal(int64(1), blobId)
		return buffer, nil
	}
	s := ProvideAsyncActualizerStateFactory()(context.Background(), mockedHostStateStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, 0, 0, WithFederationBlobHandler(federatioBlobHandler))

	k, err := s.KeyBuilder(sys.Storage_FederationBlob, appdef.NullQName)
	require.NoError(err)

	k.PutString(sys.Storage_FederationBlob_Field_Owner, "owner")
	k.PutString(sys.Storage_FederationBlob_Field_AppName, "appname")
	k.PutInt64(sys.Storage_FederationBlob_Field_BlobID, 1)
	k.PutString(sys.Storage_FederationBlob_Field_ExpectedCodes, "200,201")
	k.PutInt64(sys.Storage_FederationBlob_Field_WSID, 123)

	readBytes := make([]byte, 0)
	err = s.Read(k, func(_ istructs.IKey, value istructs.IStateValue) (err error) {
		readBytes = append(readBytes, value.AsBytes(sys.Storage_FederationBlob_Field_Body)...)
		return
	})
	require.NoError(err)
	require.Len(readBytes, 5000)
	require.Equal(byte(1), readBytes[0])
	require.Equal(byte(2), readBytes[4999])
}
