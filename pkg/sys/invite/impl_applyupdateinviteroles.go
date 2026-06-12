/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// Deprecated: superseded by asyncProjectorApplyInviteEvents. Kept for backward compatibility only.
func asyncProjectorApplyUpdateInviteRoles() istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyUpdateInviteRoles,
		Func: deprecatedNullProjector,
	}
}
