/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// Deprecated: superseded by ApplyInviteEvents. Kept for backward compatibility only.
func deprecatedNullProjector(_ istructs.IPLogEvent, _ istructs.IState, _ istructs.IIntents) error {
	return nil
}

// Deprecated: superseded by asyncProjectorApplyInviteEvents. Kept for backward compatibility only.
func asyncProjectorApplyInvitation() istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyInvitation,
		Func: deprecatedNullProjector,
	}
}
