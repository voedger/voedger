/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

package isafestatehost

import (
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/isafestate"
)

func ProvideSafeState(state state.IUnsafeState) isafestate.ISafeState {
	return provideSafeStateImpl(state)
}
