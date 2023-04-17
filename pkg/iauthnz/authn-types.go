/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iauthnz

import "github.com/voedger/voedger/pkg/istructs"

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
	QName istructs.QName

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
	QNameRoleSystem          = istructs.NewQName(istructs.SysPackage, "RoleSystem")
	QNameRoleWorkspaceOwner  = istructs.NewQName(istructs.SysPackage, "RoleWorkspaceOwner")
	QNameRoleWorkspaceDevice = istructs.NewQName(istructs.SysPackage, "RoleWorkspaceDevice")

	// assigned if additionally if WorkspaceOwner or WorkspaceDevice or ProfileOwner
	QNameRoleWorkspaceSubject = istructs.NewQName(istructs.SysPackage, "RoleWorkspaceSubject")

	// assigned if request is came to subject's profile
	QNameRoleProfileOwner = istructs.NewQName(istructs.SysPackage, "RoleProfileOwner")

	// asssigned automatically if has e.g. RoleResellersAdmin or RoleUntillPaymentsReseller
	QNameRoleWorkspaceAdmin = istructs.NewQName(istructs.SysPackage, "RoleWorkspaceAdmin")
)

var SysRoles = []istructs.QName{
	QNameRoleSystem,
	QNameRoleWorkspaceOwner,
	QNameRoleWorkspaceDevice,
	QNameRoleWorkspaceSubject,
	QNameRoleProfileOwner,
	QNameRoleWorkspaceAdmin,
}
