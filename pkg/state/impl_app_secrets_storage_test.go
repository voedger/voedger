/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/sys"
)

func TestAppSecretsStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	secret := "secret.json"
	secretBody := `{"secret":"key"}`
	sr := &isecrets.SecretReaderMock{}
	sr.On("ReadSecret", secret).Return([]byte(secretBody), nil)
	s := ProvideAsyncActualizerStateFactory()(context.Background(), nilAppStructsFunc, nil, nil, nil, sr, nil, nil, nil, 0, 0)
	kb, err := s.KeyBuilder(sys.Storage_AppSecret, appdef.NullQName)
	require.NoError(err)
	kb.PutString(sys.Storage_AppSecretField_Secret, secret)

	sv, err := s.MustExist(kb)
	require.NoError(err)

	require.Equal(secretBody, sv.AsString(""))
}
func TestAppSecretsStorage(t *testing.T) {
	t.Run("Should return error when key invalid", func(t *testing.T) {
		s := ProvideAsyncActualizerStateFactory()(context.Background(), nilAppStructsFunc, nil, nil, nil, nil, nil, nil, nil, 0, 0)
		kb, err := s.KeyBuilder(sys.Storage_AppSecret, appdef.NullQName)
		require.NoError(t, err)

		_, err = s.MustExist(kb)

		require.ErrorContains(t, err, "secret name is not specified")
	})
}
