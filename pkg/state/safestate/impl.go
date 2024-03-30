/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package safestate

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type safeState struct {
	state         state.IUnsafeState
	keyBuilders   []istructs.IStateKeyBuilder
	values        []istructs.IStateValue
	valueBuilders []istructs.IValueBuilder
}

// panics on errors
func (s *safeState) KeyBuilder(storage, entity appdef.QName) TSafeKeyBuilder {
	skb, err := s.state.KeyBuilder(storage, entity)
	if err != nil {
		panic(err)
	}
	kb := TSafeKeyBuilder(len(s.keyBuilders))
	s.keyBuilders = append(s.keyBuilders, skb)
	return kb
}

func provideSafeStateImpl(state state.IUnsafeState) ISafeState {
	return &safeState{
		state: state,
	}
}
