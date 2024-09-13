/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package isecretsimpl

import (
	"github.com/voedger/voedger/pkg/isecrets"
)

func ProvideSecretReader() isecrets.ISecretReader {
	return implSecretReader()
}
