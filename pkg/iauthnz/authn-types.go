/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iauthnz

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
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
	QName schemas.QName

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
	QNameRoleSystem          = schemas.NewQName(schemas.SysPackage, "RoleSystem")
	QNameRoleWorkspaceOwner  = schemas.NewQName(schemas.SysPackage, "RoleWorkspaceOwner")
	QNameRoleWorkspaceDevice = schemas.NewQName(schemas.SysPackage, "RoleWorkspaceDevice")

	// assigned if additionally if WorkspaceOwner or WorkspaceDevice or ProfileOwner
	QNameRoleWorkspaceSubject = schemas.NewQName(schemas.SysPackage, "RoleWorkspaceSubject")

	// assigned if request is came to subject's profile
	QNameRoleProfileOwner = schemas.NewQName(schemas.SysPackage, "RoleProfileOwner")

	// asssigned automatically if has e.g. RoleResellersAdmin or RoleUntillPaymentsReseller
	QNameRoleWorkspaceAdmin = schemas.NewQName(schemas.SysPackage, "RoleWorkspaceAdmin")
)

var SysRoles = []schemas.QName{
	QNameRoleSystem,
	QNameRoleWorkspaceOwner,
	QNameRoleWorkspaceDevice,
	QNameRoleWorkspaceSubject,
	QNameRoleProfileOwner,
	QNameRoleWorkspaceAdmin,
}
