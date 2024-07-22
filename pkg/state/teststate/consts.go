/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package teststate

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
	TestPartition = istructs.PartitionID(1)
)

var IntentsLimit = 10
var BundlesLimit = 10
var (
	AppQName_test = appdef.NewAppQName("test", "app")
	PackageName   = "tstpkg"
)
