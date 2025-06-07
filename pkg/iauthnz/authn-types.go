/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iauthnz

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type AuthnRequest struct {
	// Host from which request is issued
	// Can be empty
	Host string

	RequestWSID istructs.WSID

	Token string
}

type Principal struct {
	Kind PrincipalKindType

	// PrincipalKind_User 	- ProfileWSID
	// PrincipalKind_Group 	- Workspace this group belongs to
	// PrincipalKind_Device - ProfileWSID
	// PrincipalKind_Role   - Workspace this role belongs to
	WSID istructs.WSID

	// PrincipalKind_Host	- Host address
	// PrincipalKind_User	- Login name
	Name string

	// PrincipalKind_Role	- Role name
	QName appdef.QName

	// PrincipalKind_Group	- GroupID
	ID istructs.IDType
}

type PrincipalKindType byte

const (
	PrincipalKind_NULL PrincipalKindType = iota
	PrincipalKind_Host
	PrincipalKind_User
	PrincipalKind_Role
	PrincipalKind_Group
	PrincipalKind_Device
	PrincipalKind_FakeLast
)

var (
	QNameRoleSystem = appdef.QNameRoleSystem

	// Deprecated: use QNameRoleWorkspaceOwner. Kept for backward compatibility
	QNameRoleRoleWorkspaceOwner = appdef.NewQName(appdef.SysPackage, "RoleWorkspaceOwner")

	QNameRoleWorkspaceOwner  = appdef.NewQName(appdef.SysPackage, "WorkspaceOwner")
	QNameRoleWorkspaceDevice = appdef.NewQName(appdef.SysPackage, "WorkspaceDevice")

	// assigned if request is came to subject's profile
	QNameRoleProfileOwner = appdef.NewQName(appdef.SysPackage, "ProfileOwner")

	// asssigned automatically if has e.g. ResellersAdmin or UntillPaymentsReseller
	QNameRoleWorkspaceAdmin = appdef.NewQName(appdef.SysPackage, "WorkspaceAdmin")

	// assigned if a valid token is provided
	QNameRoleAuthenticatedUser = appdef.NewQName(appdef.SysPackage, "AuthenticatedUser")

	// assigned regardles of wether token is rpvided or not
	QNameRoleEveryone = appdef.NewQName(appdef.SysPackage, "Everyone")

	// assigned if token is not provided
	QNameRoleAnonymous = appdef.NewQName(appdef.SysPackage, "Anonymous")

	// assigned if a token is not provided
	QNameRoleGuest = appdef.NewQName(appdef.SysPackage, "Guest")
)

var SysRoles = []appdef.QName{
	QNameRoleSystem,
	QNameRoleWorkspaceOwner,
	QNameRoleRoleWorkspaceOwner,
	QNameRoleWorkspaceDevice,
	QNameRoleProfileOwner,
	QNameRoleWorkspaceAdmin,
}

var rolesInheritance = map[appdef.QName]appdef.QName{
	QNameRoleProfileOwner:       QNameRoleWorkspaceOwner,
	QNameRoleWorkspaceDevice:    QNameRoleWorkspaceOwner,
	QNameRoleRoleWorkspaceOwner: QNameRoleWorkspaceOwner,
}
