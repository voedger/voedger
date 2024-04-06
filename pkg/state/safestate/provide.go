/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

package safestate

import (
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/isafeapi"
)

func Provide(state state.IState) isafeapi.ISafeAPI {
	return &safeState{
		state: state,
	}
}
