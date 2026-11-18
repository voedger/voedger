/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package workspace

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

// This scenario double stays local because federation imports coreutils.
type mockCreationFederation struct {
	federation.IFederationWithRetry
	mock.Mock
}

func (f *mockCreationFederation) Func(path, body string, opts ...httpu.ReqOptFunc) (*federation.FuncResponse, error) {
	args := f.Called(path, body, opts)
	var response *federation.FuncResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*federation.FuncResponse)
	}
	return response, args.Error(1)
}

func TestWorkspaceCreationErrors(t *testing.T) {
	require := require.New(t)
	businessError := errors.New("creation rejected")
	ownerError := errors.New("owner unavailable")
	for _, stage := range []string{"CreateWorkspaceID", "CreateWorkspace"} {
		t.Run(stage, func(t *testing.T) {
			for _, test := range []struct {
				name                                     string
				creationError, ownerError, expectedError error
				updateOwner                              bool
			}{
				{name: "Success does not update the owner"},
				{name: "Retry exhaustion leaves the owner pending", creationError: httpu.ErrRetryExhausted, expectedError: httpu.ErrRetryExhausted},
				{name: "Wrapped retry exhaustion leaves the owner pending", creationError: fmt.Errorf("federation: %w", httpu.ErrRetryExhausted), expectedError: httpu.ErrRetryExhausted},
				{name: "Joined retry exhaustion leaves the owner pending", creationError: errors.Join(businessError, httpu.ErrRetryExhausted), expectedError: httpu.ErrRetryExhausted},
				{name: "Business failure updates the owner", creationError: businessError, updateOwner: true},
				{name: "Owner update failure is propagated", creationError: businessError, ownerError: ownerError, expectedError: ownerError, updateOwner: true},
			} {
				t.Run(test.name, func(t *testing.T) {
					fed := &mockCreationFederation{}
					fed.Test(t)
					creationPath := "api/test1/app1/200/c.sys." + stage
					ownerPath := "api/test1/app1/100/c.sys.CUD"
					fed.On("Func", creationPath, mock.Anything, mock.Anything).Return(nil, test.creationError).Once()
					if test.updateOwner {
						fed.On("Func", ownerPath, `{"cuds":[{"sys.ID":42,"fields":{"WSID":0,"WSError":"creation rejected"}}]}`, mock.Anything).Return(nil, test.ownerError).Once()
					}
					row := &coreutils.TestObject{Name: QNameCDocWorkspaceID, ID_: 42, IsNew_: true, Data: map[string]interface{}{
						authnz.Field_WSID: int64(200), authnz.Field_WSName: "child", authnz.Field_WSKind: authnz.QNameCDoc_WorkspaceKind_UserProfile,
						Field_OwnerWSID: int64(100), Field_OwnerID: int64(42), Field_OwnerApp: "test1/app1", Field_OwnerQName2: "sys.ChildWorkspace",
					}}
					tokens := itokensjwt.TestTokensJWT()
					var invoke func() error
					if stage == "CreateWorkspaceID" {
						row.Name = authnz.QNameCDocChildWorkspace
						invoke = func() error {
							return ApplyInvokeCreateWorkspaceID(fed, istructs.AppQName_test1_app1, tokens, "child", row.AsQName(authnz.Field_WSKind), 200, "test1/app1", "", "", row, 100)
						}
					} else {
						state := &coreutils.MockState{}
						state.On("App").Return(istructs.AppQName_test1_app1)
						event := &coreutils.MockPLogEvent{}
						event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) { args.Get(0).(func(istructs.ICUDRow) bool)(row) })
						projector := invokeCreateWorkspaceProjector(fed, tokens)
						invoke = func() error { return projector(event, state, nil) }
					}
					// The production branch must preserve retry errors and avoid owner mutation.
					actual := invoke()
					if test.expectedError == nil {
						require.NoError(actual)
					} else {
						require.ErrorIs(actual, test.expectedError)
						if errors.Is(test.expectedError, httpu.ErrRetryExhausted) {
							require.Equal(test.creationError, actual)
						}
					}
					if errors.Is(test.expectedError, httpu.ErrRetryExhausted) {
						// Replaying the same input must retry the command after the destination recovers.
						fed.On("Func", creationPath, mock.Anything, mock.Anything).Return(nil, nil).Once()
						require.NoError(invoke())
					}
					if !test.updateOwner {
						fed.AssertNotCalled(t, "Func", ownerPath, mock.Anything, mock.Anything)
					}
					fed.AssertExpectations(t)
				})
			}
		})
	}
}
