/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 */

package payloads

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type IAppTokensFactory interface {
	// Should be called per App partition
	New(app appdef.AppQName) istructs.IAppTokens
}
