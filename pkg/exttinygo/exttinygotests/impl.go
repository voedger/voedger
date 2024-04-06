/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/isafestatehost"
	"github.com/voedger/voedger/pkg/state/teststate"
)

func NewTestState(processorKind int, packagePath string, createWorkspaces ...teststate.TestWorkspace) teststate.ITestState {
	ts := teststate.NewTestState(processorKind, packagePath, createWorkspaces...)
	internal.State = isafestatehost.ProvideSafeState(ts)
	return ts
}
