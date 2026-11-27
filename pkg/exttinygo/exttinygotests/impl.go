/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import (
	"testing"

	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
	"github.com/voedger/voedger/pkg/state/safestate"
	"github.com/voedger/voedger/pkg/state/teststate"
)

type TestAPI struct {
	teststate.ITestAPI
	api safe.IStateSafeAPI
}

// Release releases events acquired by the test extension's single WLog lookups.
func (s *TestAPI) Release() { s.api.Release() }

// NewTestAPI provides state for direct extension calls. Register its Release method with t.Cleanup.
func NewTestAPI(processorKind int, packagePath string, createWorkspaces ...teststate.TestWorkspace) *TestAPI {
	ts := teststate.NewTestState(processorKind, packagePath, createWorkspaces...)
	api := safestate.Provide(ts, nil)
	internal.SafeStateAPI = api
	return &TestAPI{ITestAPI: ts, api: api}
}

func NewCommandTest(t *testing.T, iCommand teststate.ICommand, extensionFunc func()) *teststate.CommandTestState {
	t.Helper()
	ts := teststate.NewCommandTestState(t, iCommand, extensionFunc)
	api := safestate.Provide(ts, nil)
	t.Cleanup(api.Release)
	internal.SafeStateAPI = api
	return ts
}

func NewProjectorTest(t *testing.T, extensionFunc func()) *teststate.ProjectorTestState {
	t.Helper()
	ts := teststate.NewProjectorTestState(t, extensionFunc)
	api := safestate.Provide(ts, nil)
	t.Cleanup(api.Release)
	internal.SafeStateAPI = api
	return ts
}
