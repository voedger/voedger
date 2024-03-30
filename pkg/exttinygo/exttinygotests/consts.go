/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	ProcKind_Actualizer = iota
	ProcKind_CommandProcessor
	ProcKind_QueryProcessor
)

const (
	testPkgAlias  = "tstpkg"
	wasmFilename  = "pkg.wasm"
	testPartition = istructs.PartitionID(1)
)

var IntentsLimit = 10
var BundlesLimit = 10
var AppQName_test = istructs.NewAppQName("test", "app")
var newWorkspaceCmd = appdef.NewQName("sys", "NewWorkspace")
