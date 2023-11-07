/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package registry

import "github.com/voedger/voedger/pkg/istructs"

// for both Initiate*ResetPassword and Issue*ForResetPassword
type result struct {
	istructs.NullObject
	token       string
	profileWSID int64
}
