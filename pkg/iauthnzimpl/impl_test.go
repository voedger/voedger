/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	alienWSID                 = 3
	nonInitedWSID             = 4
	unlinkedDeviceProfileWSID = 5
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	logger.SetLogLevel(logger.LogLevelVerbose)
	defer logger.SetLogLevel(logger.LogLevelInfo)

	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(istructs.AppQName_test1_app1)
	pp := payloads.PrincipalPayload{
		Login:       "testlogin",
		SubjectKind: istructs.SubjectKind_User,
		Roles: []payloads.RoleType{
			{
				WSID:  42,
				QName: iauthnz.QNameRoleWorkspaceDevice,
			},
		},
		ProfileWSID: 1,
	}
	token, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	appStructs := AppStructsWithTestStorage(map[istructs.WSID]map[appdef.QName]map[istructs.RecordID]map[string]interface{}{
		// WSID 1 is the user profile, not necessary to store docs there

		// workspace owned by the user
		istructs.WSID(2): {
			qNameCDocWorkspaceDescriptor: {
				// cdoc.sys.WorkspaceDescriptor.ID=1, .OwnerWSID=1
				1: {
					"OwnerWSID": int64(1), // the same as ProfileWSID
				},
			},
		},

		// child workspace. Parent is WSID 2
		istructs.WSID(3): {
			qNameCDocWorkspaceDescriptor: {
				// cdoc.sys.WorkspaceDescriptor.ID=1, .OwnerWSID=2
				1: {
					"OwnerWSID": int64(2),
				},
			},
		},
	})
	authn := NewDefaultAuthenticator(TestSubjectRolesGetter)
	authz := NewDefaultAuthorizer()
	t.Run("authenticate in the profile", func(t *testing.T) {
		req := iauthnz.AuthnRequest{
			Host:        "127.0.0.1",
			RequestWSID: 1,
			Token:       token,
		}
		principals, principalPayload, err := authn.Authenticate(context.Background(), appStructs, appTokens, req)
		require.NoError(err)
		require.Len(principals, 4)
		require.Equal(iauthnz.PrincipalKind_User, principals[0].Kind)

		// request to the profile -> ProfileOwner role got
		require.Equal(iauthnz.PrincipalKind_Role, principals[1].Kind)
		require.Equal(iauthnz.QNameRoleProfileOwner, principals[1].QName)
		require.Equal(iauthnz.PrincipalKind_Role, principals[2].Kind)
		require.Equal(iauthnz.QNameRoleWorkspaceSubject, principals[2].QName)
		require.Equal(iauthnz.PrincipalKind_Host, principals[3].Kind)
		require.Equal(pp, principalPayload)
	})

	t.Run("authenticate in the owned workspace", func(t *testing.T) {
		req := iauthnz.AuthnRequest{
			Host:        "127.0.0.1",
			RequestWSID: 2,
			Token:       token,
		}
		// request to WSID 2, there is a cdoc.sys.WorkspaceDescriptor.OwnerWSID = 1 -> the workspace is owned by the user with ProfileWSID=1
		principals, principalPayload, err := authn.Authenticate(context.Background(), appStructs, appTokens, req)
		require.NoError(err)
		require.Len(principals, 4)
		require.Equal(iauthnz.PrincipalKind_User, principals[0].Kind)

		// request to the owned workspace -> WorkspaceOwner role got
		require.Equal(iauthnz.PrincipalKind_Role, principals[1].Kind)
		require.Equal(iauthnz.QNameRoleWorkspaceOwner, principals[1].QName)
		require.Equal(iauthnz.PrincipalKind_Role, principals[2].Kind)
		require.Equal(iauthnz.QNameRoleWorkspaceSubject, principals[2].QName)
		require.Equal(iauthnz.PrincipalKind_Host, principals[3].Kind)
		require.Equal(pp, principalPayload)
	})

	t.Run("authenticate in the child workspace", func(t *testing.T) {
		pp := payloads.PrincipalPayload{
			Login:       "testlogin",
			SubjectKind: istructs.SubjectKind_User,
			Roles: []payloads.RoleType{
				{
					WSID:  2,
					QName: iauthnz.QNameRoleWorkspaceOwner,
				},
			},
			ProfileWSID: 1,
		}
		token, err := appTokens.IssueToken(time.Minute, &pp)
		require.NoError(err)
		req := iauthnz.AuthnRequest{
			Host:        "127.0.0.1",
			RequestWSID: 3,
			Token:       token,
		}
		// request to WSID 2, there is a cdoc.sys.WorkspaceDescriptor.OwnerWSID = 1 -> the workspace is owned by the user with ProfileWSID=1
		principals, principalPayload, err := authn.Authenticate(context.Background(), appStructs, appTokens, req)
		require.NoError(err)
		require.Len(principals, 4)
		require.Equal(iauthnz.PrincipalKind_User, principals[0].Kind)

		// request to a workspace with a token enriched by WorkspaceOwne role -> WorkspaceOwner role got
		require.Equal(iauthnz.PrincipalKind_Role, principals[1].Kind)
		require.Equal(iauthnz.QNameRoleWorkspaceOwner, principals[1].QName)
		require.Equal(iauthnz.PrincipalKind_Role, principals[2].Kind)
		require.Equal(iauthnz.QNameRoleWorkspaceSubject, principals[2].QName)
		require.Equal(iauthnz.PrincipalKind_Host, principals[3].Kind)
		require.Equal(pp, principalPayload)
	})

	t.Run("authorize", func(t *testing.T) {
		// we are owner -> can do everything, e.g. execute sys.SomeCmd
		authnReq := iauthnz.AuthnRequest{
			Host:        "127.0.0.1",
			RequestWSID: 2,
			Token:       token,
		}
		principals, _, err := authn.Authenticate(context.Background(), appStructs, appTokens, authnReq)
		require.NoError(err)
		authzReq := iauthnz.AuthzRequest{
			OperationKind: iauthnz.OperationKind_EXECUTE,
			Resource:      appdef.NewQName(appdef.SysPackage, "SomeCmd"),
		}
		ok, err := authz.Authorize(appStructs, principals, authzReq)
		require.NoError(err)
		require.True(ok)
	})
}

func TestAuthenticate(t *testing.T) {
	require := require.New(t)

	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(istructs.AppQName_test1_app1)
	login := "testlogin"
	pp := payloads.PrincipalPayload{
		Login:       login,
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: 1,
	}
	userToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	testRole := appdef.NewQName(appdef.SysPackage, "test")
	apiKeyToken, err := IssueAPIToken(appTokens, time.Hour, []appdef.QName{
		testRole,
	}, 2, pp)
	require.NoError(err)

	pp.ProfileWSID = istructs.NullWSID
	sysToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)
	pp.ProfileWSID = 1

	pp.SubjectKind = istructs.SubjectKind_Device
	deviceToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	pp.ProfileWSID = unlinkedDeviceProfileWSID
	unlinkedDeviceToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	qNameCDocComputers := appdef.NewQName("untill", "computers")

	appStructs := AppStructsWithTestStorage(map[istructs.WSID]map[appdef.QName]map[istructs.RecordID]map[string]interface{}{
		// WSID 1 is the user profile
		istructs.WSID(1): {
			qNameViewDeviceProfileWSIDIdx: {
				1: {
					field_dummy:                 int32(1),
					field_DeviceProfileWSID:     int64(1),
					appdef.SystemField_IsActive: true,
					field_ComputersID:           istructs.RecordID(2),
					field_RestaurantComputersID: istructs.RecordID(3),
				},
				4: {
					field_dummy:                 int32(1),
					field_DeviceProfileWSID:     int64(unlinkedDeviceProfileWSID),
					appdef.SystemField_IsActive: true,
					field_ComputersID:           istructs.RecordID(5),
					field_RestaurantComputersID: istructs.RecordID(6),
				},
			},
			// wrong to store in the user profile wsid, but ok for test
			qNameCDocComputers: {
				2: {
					appdef.SystemField_QName:    qNameCDocComputers,
					appdef.SystemField_IsActive: true,
				},
				5: {
					appdef.SystemField_QName:    qNameCDocComputers,
					appdef.SystemField_IsActive: false,
				},
			},
			// not used for authorization, but keep for an example
			appdef.NewQName("untill", "restaurant_computers"): {
				3: {},
				6: {},
			},
		},

		// workspace owned by the user
		istructs.WSID(2): {
			qNameCDocWorkspaceDescriptor: {
				// cdoc.sys.WorkspaceDescriptor.ID=1, .OwnerWSID=1
				1: {
					"OwnerWSID": int64(1), // the same as ProfileWSID
				},
			},
		},

		// child workspace. Parent is WSID 2
		istructs.WSID(3): {
			qNameCDocWorkspaceDescriptor: {
				// cdoc.sys.WorkspaceDescriptor.ID=1, .OwnerWSID=2
				1: {
					"OwnerWSID": int64(2),
				},
			},
		},
	})

	testCases := []struct {
		desc               string
		req                iauthnz.AuthnRequest
		expectedPrincipals []iauthnz.Principal
		subjects           []appdef.QName
	}{
		{
			desc: "no auth -> host only",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 1,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "system token -> host and system role",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 1,
				Token:       sysToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_Role, WSID: 1, QName: iauthnz.QNameRoleSystem},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "request to profile -> host + user + profile + workspace",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 1,
				Token:       userToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_User, WSID: 1, Name: login},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 1, QName: iauthnz.QNameRoleProfileOwner},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 1, QName: iauthnz.QNameRoleWorkspaceSubject},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "request to an owned workspace -> host + user + owner + workspace",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 2,
				Token:       userToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_User, WSID: 1, Name: login},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceOwner},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceSubject},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "request to a non-owned workspace -> host + user",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: alienWSID,
				Token:       userToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_User, WSID: 1, Name: login},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "request to a non-initialized workspace -> host + user",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: nonInitedWSID,
				Token:       userToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_User, WSID: 1, Name: login},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "device -> host + device + linkedDevice + workspace",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 1,
				Token:       deviceToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_Device, WSID: 1},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 1, QName: iauthnz.QNameRoleWorkspaceDevice},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 1, QName: iauthnz.QNameRoleWorkspaceSubject},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "unlinked device -> host + device",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 1,
				Token:       unlinkedDeviceToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_Device, WSID: unlinkedDeviceProfileWSID},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
		},
		{
			desc: "ResellerAdmin -> WorkspaceAdmin",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 2,
				Token:       userToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: qNameRoleResellersAdmin},
				{Kind: iauthnz.PrincipalKind_User, WSID: 1, Name: login},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceOwner},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceSubject},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceAdmin},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
			subjects: []appdef.QName{qNameRoleResellersAdmin},
		},
		{
			desc: "UntillPaymentsReseller -> WorkspaceAdmin",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 2,
				Token:       userToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: qNameRoleUntillPaymentsReseller},
				{Kind: iauthnz.PrincipalKind_User, WSID: 1, Name: login},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceOwner},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceSubject},
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: iauthnz.QNameRoleWorkspaceAdmin},
				{Kind: iauthnz.PrincipalKind_Host, Name: "127.0.0.1"},
			},
			subjects: []appdef.QName{qNameRoleUntillPaymentsReseller},
		},
		{
			desc: "IsPersonalAccessToken -> principals are built by provided roles only",
			req: iauthnz.AuthnRequest{
				Host:        "127.0.0.1",
				RequestWSID: 2,
				Token:       apiKeyToken,
			},
			expectedPrincipals: []iauthnz.Principal{
				{Kind: iauthnz.PrincipalKind_Role, WSID: 2, QName: testRole},
			},
		},
	}
	var subjects *[]appdef.QName
	subjectsGetter := func(context.Context, string, istructs.IAppStructs, istructs.WSID) ([]appdef.QName, error) {
		return *subjects, nil
	}
	authn := NewDefaultAuthenticator(subjectsGetter)
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			subjects = &tc.subjects
			principals, _, err := authn.Authenticate(context.Background(), appStructs, appTokens, tc.req)
			require.NoError(err)
			require.Equal(tc.expectedPrincipals, principals)
		})
	}
}

func TestAuthorize(t *testing.T) {
	require := require.New(t)

	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(istructs.AppQName_test1_app1)
	pp := payloads.PrincipalPayload{
		Login:       "testlogin",
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: 1,
	}
	userToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	pp.Roles = append(pp.Roles, payloads.RoleType{
		WSID:  2,
		QName: iauthnz.QNameRoleWorkspaceSubject,
	})
	userTokenWithRole, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	pp.ProfileWSID = istructs.NullWSID
	systemToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)
	pp.ProfileWSID = 1

	pp.SubjectKind = istructs.SubjectKind_Device
	deviceToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	pp.ProfileWSID = unlinkedDeviceProfileWSID
	unlinkedDeviceToken, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(err)

	qNameCDocComputers := appdef.NewQName("untill", "computers")

	appStructs := AppStructsWithTestStorage(map[istructs.WSID]map[appdef.QName]map[istructs.RecordID]map[string]interface{}{
		// workspace owned by the user
		istructs.WSID(2): {
			qNameCDocWorkspaceDescriptor: {
				// cdoc.sys.WorkspaceDescriptor.ID=1, .OwnerWSID=1
				1: {
					"OwnerWSID": int64(1), // the same as ProfileWSID
				},
			},
			qNameViewDeviceProfileWSIDIdx: {
				2: {
					field_dummy:                 int32(1),
					field_DeviceProfileWSID:     int64(1),
					appdef.SystemField_IsActive: true,
					field_ComputersID:           istructs.RecordID(3),
					field_RestaurantComputersID: istructs.RecordID(4),
				},
				5: {
					field_dummy:                 int32(1),
					field_DeviceProfileWSID:     int64(unlinkedDeviceProfileWSID),
					appdef.SystemField_IsActive: true,
					field_ComputersID:           istructs.RecordID(6),
					field_RestaurantComputersID: istructs.RecordID(7),
				},
			},
			// wrong to store in the user profile wsid, but ok for test
			qNameCDocComputers: {
				3: {
					appdef.SystemField_QName:    qNameCDocComputers,
					appdef.SystemField_IsActive: true,
				},
				6: {
					appdef.SystemField_QName:    qNameCDocComputers,
					appdef.SystemField_IsActive: false,
				},
			},
		},

		// child workspace. Parent is WSID 2
		istructs.WSID(3): {
			qNameCDocWorkspaceDescriptor: {
				// cdoc.sys.WorkspaceDescriptor.ID=1, .OwnerWSID=2
				1: {
					"OwnerWSID": int64(2),
				},
			},
		},
	})
	authn := NewDefaultAuthenticator(TestSubjectRolesGetter)
	authz := NewDefaultAuthorizer()

	testCmd := appdef.NewQName(appdef.SysPackage, "testcmd")
	testCases := []struct {
		desc     string
		reqz     iauthnz.AuthzRequest
		reqn     iauthnz.AuthnRequest
		expected bool
	}{
		{
			desc: "execute in profile -> ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: 1,
				Token:       userToken,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      testCmd,
			},
			expected: true,
		},
		{
			desc: "execute in an owned workspace -> ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: 2,
				Token:       userToken,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      testCmd,
			},
			expected: true,
		},
		{
			desc: "execute a func with null auth policy w/o token at all -> ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: 1,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      qNameCmdLinkDeviceToRestaurant, // has null auth policy in default ACL
			},
			expected: true,
		},
		{
			desc: "execute in an alien workspace -> !ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: alienWSID,
				Token:       userToken,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      testCmd,
			},
			expected: false,
		},
		{
			desc: "execute in an alien workspace but have role saying that we are workspacesubject -> ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: alienWSID,
				Token:       userTokenWithRole,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      testCmd,
			},
			expected: true,
		},
		{
			desc: "execute in an alien workspace with system token -> ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: alienWSID,
				Token:       systemToken,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      testCmd,
			},
			expected: true,
		},
		{
			desc: "execute in an owned workspace with device token -> ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: 2,
				Token:       deviceToken,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      testCmd,
			},
			expected: true,
		},
		{
			desc: "execute in an owned workspace with an unlinked device token -> !ok",
			reqn: iauthnz.AuthnRequest{
				RequestWSID: 2,
				Token:       unlinkedDeviceToken,
			},
			reqz: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      testCmd,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			principals, _, err := authn.Authenticate(context.Background(), appStructs, appTokens, tc.reqn)
			require.NoError(err)
			ok, err := authz.Authorize(appStructs, principals, tc.reqz)
			require.NoError(err)
			require.Equal(tc.expected, ok, tc.desc)
		})
	}
}

func TestACLAllow(t *testing.T) {
	defer logger.SetLogLevel(logger.LogLevelInfo)
	require := require.New(t)
	testQName1 := appdef.NewQName(appdef.SysPackage, "testQName")

	type req struct {
		req  iauthnz.AuthzRequest
		prns [][]iauthnz.Principal
	}

	cases := []struct {
		acl  ACL
		reqs []req
	}{
		{
			acl: ACL{
				{
					desc: "allow rule",
					pattern: PatternType{
						qNamesPattern:  []appdef.QName{testQName1},
						opKindsPattern: []iauthnz.OperationKindType{iauthnz.OperationKind_INSERT},
						principalsPattern: [][]iauthnz.Principal{
							// OR
							{{Kind: iauthnz.PrincipalKind_User, Name: "testname", ID: 1, WSID: 2}},
							{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceOwner}},
						},
					},
					policy: ACPolicy_Allow,
				},
			},
			reqs: []req{
				{
					req: iauthnz.AuthzRequest{
						OperationKind: iauthnz.OperationKind_INSERT,
						Resource:      testQName1,
						Fields:        []string{"fld1", "fld2"}, // just an example
					},
					prns: [][]iauthnz.Principal{
						{
							{
								Kind: iauthnz.PrincipalKind_User,
								Name: "wrong",
							},
							{
								Kind: iauthnz.PrincipalKind_User,
								Name: "testname",
								ID:   1,
								WSID: 2,
							},
							{
								Kind: iauthnz.PrincipalKind_Host,
								Name: "127.0.0.1",
							},
							{
								Kind:  iauthnz.PrincipalKind_Role,
								QName: iauthnz.QNameRoleProfileOwner,
							},
							{
								Kind:  iauthnz.PrincipalKind_Role,
								QName: iauthnz.QNameRoleWorkspaceOwner,
							},
							{
								Kind:  iauthnz.PrincipalKind_Group,
								QName: appdef.NewQName(appdef.SysPackage, "testGroup"),
							},
							{
								Kind: iauthnz.PrincipalKind_Device,
								Name: "testDevice",
							},
						},
					},
				},
			},
		},
		{
			acl: ACL{
				{
					desc: "non-first principal in the pattern matches",
					pattern: PatternType{
						qNamesPattern: []appdef.QName{qNameCmdCreateUPProfile},
						principalsPattern: [][]iauthnz.Principal{
							// OR
							{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
							{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller, WSID: 42}},
						},
					},
					policy: ACPolicy_Allow,
				},
			},
			reqs: []req{
				{
					req: iauthnz.AuthzRequest{
						OperationKind: iauthnz.OperationKind_EXECUTE,
						Resource:      qNameCmdCreateUPProfile,
					},
					prns: [][]iauthnz.Principal{
						{
							{
								Kind:  iauthnz.PrincipalKind_Role,
								QName: qNameRoleUntillPaymentsReseller,
								WSID:  42,
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		authz := implIAuthorizer{acl: c.acl}
		for _, req := range c.reqs {
			for _, prns := range req.prns {
				ok, err := authz.Authorize(nil, prns, req.req)
				require.NoError(err)
				require.True(ok)
			}
		}
	}
}

func TestACLDeny(t *testing.T) {
	logger.SetLogLevel(logger.LogLevelVerbose)
	defer logger.SetLogLevel(logger.LogLevelInfo)
	require := require.New(t)
	testQName1 := appdef.NewQName(appdef.SysPackage, "testQName")

	type req struct {
		req  iauthnz.AuthzRequest
		prns [][]iauthnz.Principal
	}

	acl := ACL{
		{
			desc: "deny rule",
			pattern: PatternType{
				qNamesPattern:  []appdef.QName{testQName1},
				opKindsPattern: []iauthnz.OperationKindType{iauthnz.OperationKind_INSERT},
				principalsPattern: [][]iauthnz.Principal{
					// OR
					{{Kind: iauthnz.PrincipalKind_User, Name: "testnamefordeny", ID: 1, WSID: 2}},
					{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceOwner}},
				},
			},
			policy: ACPolicy_Deny,
		},
	}

	reqs := []req{
		{
			req: iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_INSERT,
				Resource:      testQName1,
				Fields:        []string{"fld1", "fld2"}, // just an example
			},
			prns: [][]iauthnz.Principal{
				{{}},
				{
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "testname",
					},
				},
				{
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "wrongName",
					},
					{
						Kind:  iauthnz.PrincipalKind_Role,
						QName: iauthnz.QNameRoleWorkspaceOwner,
					},
				},
				{
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "testname",
					},
					{
						Kind:  iauthnz.PrincipalKind_Role,
						QName: iauthnz.QNameRoleWorkspaceOwner,
					},
				},
				{
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "testname",
						ID:   1,
					},
					{
						Kind:  iauthnz.PrincipalKind_Role,
						QName: iauthnz.QNameRoleWorkspaceOwner,
					},
				},
				{
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "testname",
						WSID: 2,
					},
					{
						Kind:  iauthnz.PrincipalKind_Role,
						QName: iauthnz.QNameRoleWorkspaceOwner,
					},
				},
				{
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "testnamefordeny",
						ID:   1,
						WSID: 42,
					},
				},
				{
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "wrong",
					},
					{
						Kind: iauthnz.PrincipalKind_User,
						Name: "testnamefordeny",
						ID:   1,
						WSID: 2,
					},
					{
						Kind: iauthnz.PrincipalKind_Host,
						Name: "127.0.0.1",
					},
					{
						Kind:  iauthnz.PrincipalKind_Role,
						QName: iauthnz.QNameRoleProfileOwner,
					},
					{
						Kind:  iauthnz.PrincipalKind_Role,
						QName: iauthnz.QNameRoleWorkspaceOwner,
					},
				},
			},
		},
	}

	authz := implIAuthorizer{acl: acl}
	for _, req := range reqs {
		for _, prns := range req.prns {
			ok, err := authz.Authorize(nil, prns, req.req)
			require.NoError(err)
			require.False(ok)
		}
	}
}

func TestErrors(t *testing.T) {
	require := require.New(t)

	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(istructs.AppQName_test1_app1)

	appStructs := &implIAppStructs{}
	authn := NewDefaultAuthenticator(TestSubjectRolesGetter)

	t.Run("wrong token", func(t *testing.T) {
		req := iauthnz.AuthnRequest{
			RequestWSID: 1,
			Token:       "wrong token",
		}
		_, _, err := authn.Authenticate(context.Background(), appStructs, appTokens, req)
		require.Error(err)
	})

	t.Run("unsupported subject kind", func(t *testing.T) {

		pp := payloads.PrincipalPayload{
			Login:       "testlogin",
			SubjectKind: istructs.SubjectKind_FakeLast,
			ProfileWSID: 1,
		}
		token, err := appTokens.IssueToken(time.Minute, &pp)
		require.NoError(err)
		req := iauthnz.AuthnRequest{
			RequestWSID: 1,
			Token:       token,
		}
		_, _, err = authn.Authenticate(context.Background(), appStructs, appTokens, req)
		require.Error(err)
	})

	t.Run("personal access token for a system role", func(t *testing.T) {
		for _, sysRole := range iauthnz.SysRoles {
			token, err := IssueAPIToken(appTokens, time.Hour, []appdef.QName{sysRole}, 1, payloads.PrincipalPayload{})
			require.ErrorIs(err, ErrPersonalAccessTokenOnSystemRole)
			require.Empty(token)
		}
	})

	t.Run("personal access token for NullWSID", func(t *testing.T) {
		token, err := IssueAPIToken(appTokens, time.Hour, []appdef.QName{appdef.NewQName(appdef.SysPackage, "test")}, istructs.NullWSID, payloads.PrincipalPayload{})
		require.ErrorIs(err, ErrPersonalAccessTokenOnNullWSID)
		require.Empty(token)
	})
}

// with principals cache:  1455242       782.8 ns/op	     432 B/op	       9 allocs/op
// without principals cache: 45534	     24370 ns/op	    7964 B/op	     126 allocs/op
func BenchmarkBasic(b *testing.B) {
	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(istructs.AppQName_test1_app1)
	pp := payloads.PrincipalPayload{
		Login:       "testlogin",
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: 1,
	}
	token, err := appTokens.IssueToken(time.Minute, &pp)
	require.NoError(b, err)
	var principals []iauthnz.Principal
	appStructs := &implIAppStructs{}
	authn := NewDefaultAuthenticator(TestSubjectRolesGetter)
	authz := NewDefaultAuthorizer()
	reqn := iauthnz.AuthnRequest{
		Host:        "127.0.0.1",
		RequestWSID: 1,
		Token:       token,
	}
	reqz := iauthnz.AuthzRequest{
		OperationKind: iauthnz.OperationKind_EXECUTE,
		Resource:      appdef.NewQName(appdef.SysPackage, "SomeCmd"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		principals, _, err = authn.Authenticate(context.Background(), appStructs, appTokens, reqn)
		if err != nil {
			b.Fatal()
		}
		ok, err := authz.Authorize(appStructs, principals, reqz)
		if !ok || err != nil {
			b.Fatal()
		}
	}
}

func AppStructsWithTestStorage(data map[istructs.WSID]map[appdef.QName]map[istructs.RecordID]map[string]interface{}) istructs.IAppStructs {
	recs := &implIRecords{data: data}
	return &implIAppStructs{records: recs, views: &implIViewRecords{records: recs}}
}

type implIAppStructs struct {
	records *implIRecords
	views   *implIViewRecords
}

func (as *implIAppStructs) Events() istructs.IEvents            { panic("") }
func (as *implIAppStructs) Records() istructs.IRecords          { return as.records }
func (as *implIAppStructs) ViewRecords() istructs.IViewRecords  { return as.views }
func (as *implIAppStructs) Resources() istructs.IResources      { panic("") }
func (as *implIAppStructs) Schemas() appdef.SchemaCache         { panic("") }
func (as *implIAppStructs) ClusterAppID() istructs.ClusterAppID { panic("") }
func (as *implIAppStructs) AppQName() istructs.AppQName         { panic("") }
func (as *implIAppStructs) IsFunctionRateLimitsExceeded(appdef.QName, istructs.WSID) bool {
	panic("")
}
func (as *implIAppStructs) DescribePackageNames() []string               { panic("") }
func (as *implIAppStructs) DescribePackage(string) interface{}           { panic("") }
func (as *implIAppStructs) Uniques() istructs.IUniques                   { panic("") }
func (as *implIAppStructs) SyncProjectors() []istructs.ProjectorFactory  { panic("") }
func (as *implIAppStructs) AsyncProjectors() []istructs.ProjectorFactory { panic("") }
func (as *implIAppStructs) CUDValidators() []istructs.CUDValidator       { panic("") }
func (as *implIAppStructs) EventValidators() []istructs.EventValidator   { panic("") }
func (as *implIAppStructs) WSAmount() istructs.AppWSAmount               { panic("") }
func (as *implIAppStructs) AppTokens() istructs.IAppTokens               { panic("") }

type implIRecords struct {
	data map[istructs.WSID]map[appdef.QName]map[istructs.RecordID]map[string]interface{}
}

func (r *implIRecords) Apply(event istructs.IPLogEvent) (err error) { panic("") }
func (r *implIRecords) Apply2(event istructs.IPLogEvent, cb func(r istructs.IRecord)) (err error) {
	panic("")
}
func (r *implIRecords) Get(wsid istructs.WSID, _ bool, id istructs.RecordID) (record istructs.IRecord, err error) {
	if wsData, ok := r.data[wsid]; ok {
		for qName, qNameRecs := range wsData {
			for recID, recData := range qNameRecs {
				if recID == id {
					return &implIRecord{TestObject: coreutils.TestObject{Data: recData}, qName: qName}, nil
				}
			}
		}
	}
	return istructsmem.NewNullRecord(id), nil
}
func (r *implIRecords) GetBatch(workspace istructs.WSID, highConsistency bool, ids []istructs.RecordGetBatchItem) (err error) {
	panic("")
}
func (r *implIRecords) GetSingleton(wsid istructs.WSID, qName appdef.QName) (record istructs.IRecord, err error) {
	if wsData, ok := r.data[wsid]; ok {
		if qNameRecs, ok := wsData[qName]; ok {
			if len(qNameRecs) > 1 {
				panic(">1 records for a signleton")
			}
			for _, data := range qNameRecs {
				return &implIRecord{qName: qName, TestObject: coreutils.TestObject{Data: data}}, nil
			}
		}
	}
	return istructsmem.NewNullRecord(istructs.NullRecordID), nil
}
func (r *implIRecords) Read(workspace istructs.WSID, highConsistency bool, id istructs.RecordID) (record istructs.IRecord, err error) {
	panic("")
}

type implIRecord struct {
	coreutils.TestObject
	qName appdef.QName
}

func (r *implIRecord) QName() appdef.QName       { return r.qName }
func (r *implIRecord) ID() istructs.RecordID     { return r.AsRecordID(appdef.SystemField_ID) }
func (r *implIRecord) Parent() istructs.RecordID { panic("") }
func (r *implIRecord) Container() string         { panic("") }
func (r *implIRecord) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {
	panic("")
}
func (r *implIRecord) FieldNames(cb func(fieldName string)) { r.TestObject.FieldNames(cb) }

type implIViewRecords struct {
	records *implIRecords
}

func (vr *implIViewRecords) KeyBuilder(view appdef.QName) istructs.IKeyBuilder {
	return &implIKeyBuilder{qName: view, TestObject: coreutils.TestObject{Data: map[string]interface{}{}}}
}
func (vr *implIViewRecords) NewValueBuilder(view appdef.QName) istructs.IValueBuilder { panic("") }
func (vr *implIViewRecords) UpdateValueBuilder(view appdef.QName, existing istructs.IValue) istructs.IValueBuilder {
	panic("")
}
func (vr *implIViewRecords) Put(workspace istructs.WSID, key istructs.IKeyBuilder, value istructs.IValueBuilder) (err error) {
	panic("")
}
func (vr *implIViewRecords) PutBatch(workspace istructs.WSID, batch []istructs.ViewKV) (err error) {
	panic("")
}
func (vr *implIViewRecords) Get(workspace istructs.WSID, key istructs.IKeyBuilder) (value istructs.IValue, err error) {
	panic("")
}
func (vr *implIViewRecords) GetBatch(workspace istructs.WSID, kv []istructs.ViewRecordGetBatchItem) (err error) {
	if wsData, ok := vr.records.data[workspace]; ok {
		for biIdx, bi := range kv {
			kb := bi.Key.(*implIKeyBuilder)
			if qNameRecs, ok := wsData[kb.qName]; ok {
				for _, qNameRec := range qNameRecs {
					matchedFields := 0
					for keyField, keyValue := range kb.Data {
						if recFieldValue, ok := qNameRec[keyField]; ok {
							if recFieldValue == keyValue {
								matchedFields++
							}
						}
					}
					if len(kb.Data) == matchedFields {
						kv[biIdx].Ok = true
						kv[biIdx].Value = &implIValue{TestObject: coreutils.TestObject{Data: qNameRec}}
						break
					}
				}
			}
		}
	}
	return nil
}
func (vr *implIViewRecords) Read(ctx context.Context, workspace istructs.WSID, key istructs.IKeyBuilder, cb istructs.ValuesCallback) (err error) {
	panic("")
}

type implIKeyBuilder struct {
	coreutils.TestObject
	qName appdef.QName
}

func (kb *implIKeyBuilder) PartitionKey() istructs.IRowWriter      { return &kb.TestObject }
func (kb *implIKeyBuilder) ClusteringColumns() istructs.IRowWriter { return &kb.TestObject }
func (kb *implIKeyBuilder) Equals(istructs.IKeyBuilder) bool       { panic("implement me") }

type implIValue struct {
	coreutils.TestObject
}

func (v *implIValue) AsRecord(name string) (record istructs.IRecord) { panic("") }
func (v *implIValue) AsEvent(name string) (event istructs.IDbEvent)  { panic("") }
