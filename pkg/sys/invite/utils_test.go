/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package invite

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

var (
	testWSName      = appdef.NewQName("test", "ws")
	testWS2Name     = appdef.NewQName("other", "ws2")
	testChildWSName = appdef.NewQName("test", "childWS")
)

type testApp struct {
	ws, ws2, childWS appdef.IWorkspace
}

func buildTestApp(t *testing.T) testApp {
	t.Helper()
	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	adb.AddPackage("other", "test.com/other")

	wsb := adb.AddWorkspace(testWSName)
	wsb.AddRole(appdef.NewQName("test", "Reader"))
	wsb.AddRole(appdef.NewQName("test", "Writer"))
	wsb.AddCDoc(appdef.NewQName("test", "NotARole"))

	wsb2 := adb.AddWorkspace(testWS2Name)
	wsb2.AddRole(appdef.NewQName("other", "Admin"))

	childWSB := adb.AddWorkspace(testChildWSName)
	childWSB.SetAncestors(testWSName)
	childWSB.AddRole(appdef.NewQName("test", "ChildOnly"))

	app, err := adb.Build()
	require.NoError(t, err)
	return testApp{
		ws:      app.Workspace(testWSName),
		ws2:     app.Workspace(testWS2Name),
		childWS: app.Workspace(testChildWSName),
	}
}

func requireRolesError(t *testing.T, err error, expectedErr error) {
	t.Helper()
	require := require.New(t)
	var sysErr coreutils.SysError
	require.ErrorAs(err, &sysErr)
	require.Equal(http.StatusBadRequest, sysErr.HTTPStatus)
	require.Contains(sysErr.Message, expectedErr.Error())
}

func TestValidateInviteRoles(t *testing.T) {
	ta := buildTestApp(t)
	ws, ws2, childWS := ta.ws, ta.ws2, ta.childWS

	t.Run("happy path", func(t *testing.T) {
		t.Run("single valid role", func(t *testing.T) {
			require := require.New(t)
			require.NoError(validateInviteRoles("test.Reader", ws))
		})
		t.Run("multiple valid roles", func(t *testing.T) {
			require := require.New(t)
			require.NoError(validateInviteRoles("test.Reader,test.Writer", ws))
		})
		t.Run("spaces around commas trimmed", func(t *testing.T) {
			require := require.New(t)
			require.NoError(validateInviteRoles(" test.Reader , test.Writer ", ws))
		})
		t.Run("inherited role from ancestor", func(t *testing.T) {
			require := require.New(t)
			require.NoError(validateInviteRoles("test.Reader", childWS))
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("empty string", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("", ws), ErrRolesEmpty)
		})
		t.Run("whitespace only", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("  ", ws), ErrRolesEmpty)
		})
		t.Run("trailing comma produces empty element", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("test.Reader,", ws), ErrRolesEmpty)
		})
		t.Run("leading comma produces empty element", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles(",test.Reader", ws), ErrRolesEmpty)
		})
		t.Run("malformed QName", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("notAQName", ws), ErrRoleInvalid)
		})
		t.Run("malformed QName among valid", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("test.Reader,bad", ws), ErrRoleInvalid)
		})
		t.Run("dot only QName", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles(".", ws), ErrRoleNotFound)
		})
		t.Run("sys role rejected", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("sys.WorkspaceOwner", ws), ErrSystemRole)
		})
		t.Run("sys role among valid rejected", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("test.Reader,sys.WorkspaceAdmin", ws), ErrSystemRole)
		})
		t.Run("any sys role rejected", func(t *testing.T) {
			for _, sysRole := range []string{
				"sys.Everyone",
				"sys.Anonymous",
				"sys.AuthenticatedUser",
				"sys.System",
				"sys.ProfileOwner",
				"sys.WorkspaceDevice",
				"sys.WorkspaceOwner",
				"sys.ClusterAdmin",
				"sys.WorkspaceAdmin",
				"sys.BLOBUploader",
				"sys.RoleWorkspaceOwner",
				"sys.FutureRole",
			} {
				t.Run(sysRole, func(t *testing.T) {
					requireRolesError(t, validateInviteRoles(sysRole, ws), ErrSystemRole)
				})
			}
		})
		t.Run("role not found in workspace", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("test.NonExistent", ws), ErrRoleNotFound)
		})
		t.Run("role exists in other workspace but not in target", func(t *testing.T) {
			require := require.New(t)
			require.NotNil(appdef.Role(ws2.Type, appdef.NewQName("other", "Admin")))
			requireRolesError(t, validateInviteRoles("other.Admin", ws), ErrRoleNotFound)
		})
		t.Run("non-role type with same QName rejected", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("test.NotARole", ws), ErrRoleNotFound)
		})
		t.Run("child-only role not found in parent", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("test.ChildOnly", ws), ErrRoleNotFound)
		})
		t.Run("duplicate roles rejected", func(t *testing.T) {
			requireRolesError(t, validateInviteRoles("test.Reader,test.Reader", ws), ErrRoleDuplicate)
		})
	})
}

func TestResolveControllingInviteRejectsMultipleFallbackCandidates(t *testing.T) {
	const (
		subjectID      = istructs.RecordID(101)
		canonicalID    = istructs.RecordID(201)
		aliasID        = istructs.RecordID(202)
		excludedID     = istructs.RecordID(203)
		canonicalLogin = "jsmith@example.com"
		activeAlias    = "j.smith@example.com"
	)

	newIndexKey := func(email string) *coreutils.MockStateKeyBuilder {
		key := &coreutils.MockStateKeyBuilder{}
		key.On("PutInt32", field_Dummy, value_Dummy_One).Return().Once()
		key.On("PutString", Field_Login, email).Return().Once()
		return key
	}
	newInviteKey := func(id istructs.RecordID) *coreutils.MockStateKeyBuilder {
		key := &coreutils.MockStateKeyBuilder{}
		key.On("PutRecordID", sys.Storage_Record_Field_ID, id).Return().Once()
		return key
	}
	newJoinedInvite := func() *coreutils.MockStateValue {
		invite := &coreutils.MockStateValue{}
		invite.On("AsInt32", Field_State).Return(int32(State_Joined)).Once()
		invite.On("AsRecordID", field_SubjectID).Return(subjectID).Once()
		return invite
	}

	canonicalIndexKey := newIndexKey(canonicalLogin)
	aliasIndexKey := newIndexKey(activeAlias)
	canonicalInviteKey := newInviteKey(canonicalID)
	aliasInviteKey := newInviteKey(aliasID)

	canonicalIndex := &coreutils.MockStateValue{}
	canonicalIndex.On("AsRecordID", field_InviteID).Return(canonicalID).Once()
	aliasIndex := &coreutils.MockStateValue{}
	aliasIndex.On("AsRecordID", field_InviteID).Return(aliasID).Once()
	canonicalInvite := newJoinedInvite()
	aliasInvite := newJoinedInvite()

	subject := &coreutils.MockStateValue{}
	subject.On("AsString", Field_InviteEmail).Return("").Once()

	st := &coreutils.MockState{}
	st.On("KeyBuilder", sys.Storage_View, qNameViewInviteIndex).Return(canonicalIndexKey, nil).Once()
	st.On("CanExist", canonicalIndexKey).Return(canonicalIndex, true, nil).Once()
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocInvite).Return(canonicalInviteKey, nil).Once()
	st.On("CanExist", canonicalInviteKey).Return(canonicalInvite, true, nil).Once()
	st.On("KeyBuilder", sys.Storage_View, qNameViewInviteIndex).Return(aliasIndexKey, nil).Once()
	st.On("CanExist", aliasIndexKey).Return(aliasIndex, true, nil).Once()
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocInvite).Return(aliasInviteKey, nil).Once()
	st.On("CanExist", aliasInviteKey).Return(aliasInvite, true, nil).Once()

	inviteID, invite, err := resolveControllingInvite(subjectID, subject, canonicalLogin, activeAlias, excludedID, st)
	require.ErrorIs(t, err, ErrControllingInviteNotIdentified)
	require.Equal(t, istructs.NullRecordID, inviteID)
	require.Nil(t, invite)

	st.AssertExpectations(t)
	canonicalIndexKey.AssertExpectations(t)
	aliasIndexKey.AssertExpectations(t)
	canonicalInviteKey.AssertExpectations(t)
	aliasInviteKey.AssertExpectations(t)
	canonicalIndex.AssertExpectations(t)
	aliasIndex.AssertExpectations(t)
	canonicalInvite.AssertExpectations(t)
	aliasInvite.AssertExpectations(t)
	subject.AssertExpectations(t)
}
