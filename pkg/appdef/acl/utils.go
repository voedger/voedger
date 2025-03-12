/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func logVerboseDenyReason(op appdef.OperationKind, resource appdef.QName, failedField appdef.FieldName, roles []appdef.QName, ws appdef.IWorkspace) {
	entity := resource.String()
	if failedField != "" {
		entity += "." + failedField
	}
	logger.Verbose(fmt.Sprintf("ws %s: %s on %s by %s -> deny", ws.Descriptor(), op, entity, roles))
}
