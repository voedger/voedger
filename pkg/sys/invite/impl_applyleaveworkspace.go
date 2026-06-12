/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// Deprecated: superseded by asyncProjectorApplyInviteEvents. Kept for backward compatibility only.
func asyncProjectorApplyLeaveWorkspace() istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyLeaveWorkspace,
		Func: deprecatedNullProjector,
	}
}
