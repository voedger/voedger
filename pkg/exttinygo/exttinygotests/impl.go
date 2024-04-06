/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/safestate"
	"github.com/voedger/voedger/pkg/state/teststate"
)

func NewTestAPI(processorKind int, packagePath string, createWorkspaces ...teststate.TestWorkspace) teststate.ITestAPI {
	ts := teststate.NewTestState(processorKind, packagePath, createWorkspaces...)
	internal.State = safestate.Provide(ts)
	return ts
}
