/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package teststate

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type NewEventCallback func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD)
type ViewValueCallback func(key istructs.IKeyBuilder, value istructs.IValueBuilder)
type KeyBuilderCallback func(key istructs.IStateKeyBuilder)
type ValueBuilderCallback func(value istructs.IStateValueBuilder)

type ITestState interface {
	state.IUnsafeState
	// State
	PutEvent(wsid istructs.WSID, name appdef.FullQName, cb NewEventCallback)
	PutView(testWSID istructs.WSID, entity appdef.FullQName, callback ViewValueCallback)
	PutSecret(name string, secret []byte)

	// Intent
	RequireIntent(t *testing.T, storage appdef.QName, entity appdef.FullQName, kb KeyBuilderCallback) IIntentAssertions
}

type IIntentAssertions interface {
	Exists()
	Equal(vb ValueBuilderCallback)
}
