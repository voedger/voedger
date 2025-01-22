/*
 * Copyright (c) 2025-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package in10nmem

import (
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

func logVerbose(subject string, pkey in10n.ProjectionKey, offset istructs.Offset) {
	msg := fmt.Sprintf("%s, %s, %s, %d, %d", subject, pkey.App, pkey.Projection, pkey.WS, offset)
	logger.Verbose(msg)
}
