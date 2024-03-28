/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginetestctx

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type IExtTestContext interface {
	// State
	PutEvent(wsid istructs.WSID, name appdef.FullQName, cb NewEventCallback)
	PutView(testWSID istructs.WSID, entity appdef.FullQName, callback ViewValueCallback)
	PutSecret(name string, secret []byte)

	// Invoke
	Invoke(Extension appdef.FullQName) error

	// Intent
	HasIntent(storage appdef.QName, entity appdef.FullQName, callback HasIntentCallback) bool

	// Close
	Close()
}
