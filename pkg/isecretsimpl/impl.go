/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package isecretsimpl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/voedger/voedger/pkg/isecrets"
)

func implSecretReader() isecrets.ISecretReader {
	secretRoot := defaultSecretRoot
	if customSecretRoot := os.Getenv(SecretRootEnv); customSecretRoot != "" {
		secretRoot = customSecretRoot
	}
	return &secretReader{secretRoot: secretRoot}
}

type secretReader struct {
	secretRoot string
}

func (r *secretReader) ReadSecret(name string) (bb []byte, err error) {
	if strings.TrimSpace(name) == "" {
		return nil, isecrets.ErrSecretNameIsBlank
	}

	return os.ReadFile(filepath.Join(r.secretRoot, name))
}
