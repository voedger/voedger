/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package teststate

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type NewEventCallback func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD)
type ViewValueCallback func(key istructs.IKeyBuilder, value istructs.IValueBuilder)
type KeyBuilderCallback func(key istructs.IStateKeyBuilder)
type ValueBuilderCallback func(value istructs.IStateValueBuilder)
type HttpHandlerFunc func(req HttpRequest) (resp HttpResponse, err error)

type ITestAPI interface {
	// State
	PutEvent(wsid istructs.WSID, name appdef.FullQName, cb NewEventCallback) (wLogOffs istructs.Offset, newRecordIds []istructs.RecordID)
	PutView(testWSID istructs.WSID, entity appdef.FullQName, callback ViewValueCallback)
	PutSecret(name string, secret []byte)
	PutHttpHandler(HttpHandlerFunc)
	PutRequestSubject(principals []iauthnz.Principal, token string)

	// Intent
	RequireIntent(t *testing.T, storage appdef.QName, entity appdef.FullQName, kb KeyBuilderCallback) IIntentAssertions
}

type ITestState interface {
	state.IState
	ITestAPI
}

type IIntentAssertions interface {
	Exists()
	Equal(vb ValueBuilderCallback)
}
