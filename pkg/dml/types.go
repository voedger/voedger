/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package dml

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type OpKind int

type Workspace struct {
	ID   uint64
	Kind WorkspaceKind
}

type WorkspaceKind int

type Op struct {
	AppQName              appdef.AppQName
	QName                 appdef.QName
	Kind                  OpKind
	Workspace             Workspace
	EntityID              istructs.IDType // offset or RecordID
	CleanSQL              string
	VSQLWithoutAppAndWSID string // need to forward the query to the target app and\or WSID
}
