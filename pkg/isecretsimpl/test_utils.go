/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package isecretsimpl

import "github.com/voedger/voedger/pkg/itokensjwt"

var TestSecretReader = itokensjwt.ProvideTestSecretsReader(ProvideSecretReader())
