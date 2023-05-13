/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package builtin

import "github.com/voedger/voedger/pkg/istructs"

type echoRR struct {
	istructs.NullObject
	text string
}
