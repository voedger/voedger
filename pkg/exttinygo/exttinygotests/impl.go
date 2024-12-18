/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import (
	"testing"

	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/safestate"
	"github.com/voedger/voedger/pkg/state/teststate"
)

func NewTestAPI(processorKind int, packagePath string, createWorkspaces ...teststate.TestWorkspace) teststate.ITestAPI {
	ts := teststate.NewTestState(processorKind, packagePath, createWorkspaces...)
	internal.SafeStateAPI = safestate.Provide(ts, nil)
	return ts
}

func NewCommandTest(t *testing.T, iCommand teststate.ICommand, extensionFunc func()) *teststate.CommandTestState {
	ts := teststate.NewCommandTestState(t, iCommand, extensionFunc)
	internal.SafeStateAPI = safestate.Provide(ts, nil)
	return ts
}

func NewProjectorTest(t *testing.T, extensionFunc func()) *teststate.ProjectorTestState {
	ts := teststate.NewProjectorTestState(t, extensionFunc)
	internal.SafeStateAPI = safestate.Provide(ts, nil)
	return ts
}
