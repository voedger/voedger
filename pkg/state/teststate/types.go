/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package teststate

import "github.com/voedger/voedger/pkg/istructs"

type TestWorkspace struct {
	WorkspaceDescriptor string
	WSID                istructs.WSID
}

type TestViewValue struct {
	wsid istructs.WSID
	vr   istructs.IViewRecords
	Key  istructs.IKeyBuilder
	Val  istructs.IValueBuilder
}
