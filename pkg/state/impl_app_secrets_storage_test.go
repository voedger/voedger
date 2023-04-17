/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestAppSecretsStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	secret := "secret.json"
	secretBody := `{"secret":"key"}`
	sr := &isecrets.SecretReaderMock{}
	sr.On("ReadSecret", secret).Return([]byte(secretBody), nil)
	s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, sr, 0, 0)
	kb, err := s.KeyBuilder(AppSecretsStorage, istructs.NullQName)
	require.NoError(err)
	kb.PutString(Field_Secret, secret)

	sv, err := s.MustExist(kb)
	require.NoError(err)

	require.Equal(secretBody, sv.AsString(""))
	json, err := sv.ToJSON()
	require.NoError(err)
	require.Equal(`{"Body":"{"secret":"key"}"}`, json)
}
func TestAppSecretsStorage(t *testing.T) {
	t.Run("Should return error when key invalid", func(t *testing.T) {
		s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, 0, 0)
		kb, err := s.KeyBuilder(AppSecretsStorage, istructs.NullQName)
		require.NoError(t, err)

		_, err = s.MustExist(kb)

		require.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("Should return value that not exists when", func(t *testing.T) {
		tests := []struct {
			err error
		}{
			{
				err: isecrets.ErrSecretNameIsBlank,
			},
			{
				err: fs.ErrNotExist,
			},
		}
		for _, test := range tests {
			t.Run(test.err.Error(), func(t *testing.T) {
				sr := &isecrets.SecretReaderMock{}
				sr.On("ReadSecret", mock.Anything).Return(nil, test.err)
				s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, sr, 0, 0)
				kb, err := s.KeyBuilder(AppSecretsStorage, istructs.NullQName)
				require.NoError(t, err)
				kb.PutString(Field_Secret, "")

				_, ok, err := s.CanExist(kb)
				require.NoError(t, err)

				require.False(t, ok)
			})
		}
	})
}
