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
	QNameRoleSystem          = appdef.NewQName(appdef.SysPackage, "RoleSystem")
	QNameRoleWorkspaceOwner  = appdef.NewQName(appdef.SysPackage, "RoleWorkspaceOwner")
	QNameRoleWorkspaceDevice = appdef.NewQName(appdef.SysPackage, "RoleWorkspaceDevice")

	// assigned if additionally if WorkspaceOwner or WorkspaceDevice or ProfileOwner
	QNameRoleWorkspaceSubject = appdef.NewQName(appdef.SysPackage, "RoleWorkspaceSubject")

	// assigned if request is came to subject's profile
	QNameRoleProfileOwner = appdef.NewQName(appdef.SysPackage, "RoleProfileOwner")

	// asssigned automatically if has e.g. RoleResellersAdmin or RoleUntillPaymentsReseller
	QNameRoleWorkspaceAdmin = appdef.NewQName(appdef.SysPackage, "RoleWorkspaceAdmin")
)

var SysRoles = []appdef.QName{
	QNameRoleSystem,
	QNameRoleWorkspaceOwner,
	QNameRoleWorkspaceDevice,
	QNameRoleWorkspaceSubject,
	QNameRoleProfileOwner,
	QNameRoleWorkspaceAdmin,
}
