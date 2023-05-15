/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package verifier

import (
	"github.com/voedger/voedger/pkg/istructs"
)

type ievResult struct {
	istructs.NullObject
	verificationToken string
}

type ivvtResult struct {
	istructs.NullObject
	verifiedValueToken string
}
