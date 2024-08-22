/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestAppSecretsStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	secret := "secret.json"
	secretBody := `{"secret":"key"}`
	sr := &isecrets.SecretReaderMock{}
	sr.On("ReadSecret", secret).Return([]byte(secretBody), nil)
	storage := NewAppSecretsStorage(sr)
	kb := storage.NewKeyBuilder(appdef.NullQName, nil)
	kb.PutString(sys.Storage_AppSecretField_Secret, secret)
	sv, err := storage.(state.IWithGet).Get(kb)
	require.NoError(err)
	require.Equal(secretBody, sv.AsString(""))
}
func TestAppSecretsStorage(t *testing.T) {
	t.Run("Should return error when key invalid", func(t *testing.T) {
		storage := NewAppSecretsStorage(nil)
		kb := storage.NewKeyBuilder(appdef.NullQName, nil)
		_, err := storage.(state.IWithGet).Get(kb)
		require.ErrorContains(t, err, "secret name is not specified")
	})
}
