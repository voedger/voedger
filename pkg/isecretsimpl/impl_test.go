/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package isecretsimpl

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/isecrets"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	dir, err := os.MkdirTemp("", "isecretsimpl")
	require.NoError(err)
	require.NoError(os.Setenv(SecretRootEnv, dir))
	defer func() {
		require.NoError(os.RemoveAll(dir))
		require.NoError(os.Unsetenv(SecretRootEnv))
	}()
	secret := "secret.json"
	require.NoError(os.WriteFile(filepath.Join(dir, secret), []byte(`{"secret":"key"}`), fs.ModePerm))
	sr := ProvideSecretReader()

	bb, err := sr.ReadSecret(secret)
	require.NoError(err)
	bb1, err1 := sr.ReadSecret("")
	bb2, err2 := sr.ReadSecret(" ")
	bb3, err3 := sr.ReadSecret("file-not-exist")

	require.JSONEq(`{"secret":"key"}`, string(bb))
	require.ErrorIs(err1, isecrets.ErrSecretNameIsBlank)
	require.ErrorIs(err2, isecrets.ErrSecretNameIsBlank)
	require.ErrorIs(err3, fs.ErrNotExist)
	require.Nil(bb1)
	require.Nil(bb2)
	require.Nil(bb3)
}
