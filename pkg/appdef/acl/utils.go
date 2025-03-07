/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"fmt"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

// here to avid memory consumption for returning []allowedField and []effectiveRole
func logVerboseDenyReason(op appdef.OperationKind, resource appdef.QName, allowed []appdef.FieldName, requestedFields []string, roles []appdef.QName, ws appdef.IWorkspace) {
	entity := resource.String()
	for _, reqField := range requestedFields {
		if !slices.Contains(allowed, reqField) {
			entity += "." + reqField
			break
		}
	}
	logger.Verbose(fmt.Sprintf("ws %s: %s on %s by %s -> deny", ws.Descriptor(), op, entity, roles))
}
