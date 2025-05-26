/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import "github.com/voedger/voedger/pkg/appdef"

func isValidInviteState(state int32, cmd appdef.QName) bool {
	return inviteValidStates[cmd][State(state)]
}
