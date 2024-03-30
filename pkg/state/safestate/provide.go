/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

package safestate

import "github.com/voedger/voedger/pkg/state"

func ProvideSafeState(state state.IUnsafeState) ISafeState {
	return provideSafeStateImpl(state)
}
