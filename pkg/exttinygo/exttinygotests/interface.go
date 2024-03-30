/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type IExtTestContext interface {
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
