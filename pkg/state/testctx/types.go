/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package istatetestctx

import "github.com/voedger/voedger/pkg/istructs"

type TestWorkspace struct {
	WorkspaceDescriptor string
	WSID                istructs.WSID
}

type NewEventCallback func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD)

type TestViewValue struct {
	wsid istructs.WSID
	vr   istructs.IViewRecords
	Key  istructs.IKeyBuilder
	Val  istructs.IValueBuilder
}

type ViewValueCallback func(key istructs.IKeyBuilder, value istructs.IValueBuilder)
type HasIntentCallback func(key istructs.IStateKeyBuilder, value istructs.IStateValueBuilder)
