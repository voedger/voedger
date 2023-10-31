/*
 * Copyright (c) 2020-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package vvm

import (
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

func ProvideTestSecretsReader(realSecretsReader isecrets.ISecretReader) isecrets.ISecretReader {
	return &testISecretReader{realSecretReader: realSecretsReader}
}

type testISecretReader struct {
	realSecretReader isecrets.ISecretReader
}

func (tsr *testISecretReader) ReadSecret(name string) ([]byte, error) {
	if name == SecretKeyJWTName {
		return itokensjwt.SecretKeyExample, nil
	}
	return tsr.realSecretReader.ReadSecret(name)
}
