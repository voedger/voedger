/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 */

package payloads

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	itokens "github.com/voedger/voedger/pkg/itokens"
)

// Principal can be referenced by WSID
// Owner is a record with {WSID, IDOfOwner} key
// isAPIToken -> principals will be built by Roles only in authenticator
type PrincipalPayload struct {
	Login       string
	SubjectKind istructs.SubjectKindType
	ProfileWSID istructs.WSID
	Roles       []RoleType
	GlobalRoles []appdef.QName
	IsAPIToken  bool
}

type RoleType struct {
	// for role must be OwnerWSID, not the request WSID
	WSID istructs.WSID
	// E.g. air.LinkedDevice
	QName appdef.QName
}

type BLOBUploadingPayload struct {
	Workspace istructs.WSID
	BLOB      istructs.RecordID
	MaxSize   int64
}

type VerifiedValuePayload struct {
	VerificationKind appdef.VerificationKind
	WSID             istructs.WSID
	ID               istructs.RecordID
	Entity           appdef.QName
	Field            string
	Value            interface{}
}

type VerificationPayload struct {
	VerifiedValuePayload
	Hash256 [32]byte
}

type implIAppTokensFactory struct {
	tokens itokens.ITokens
}

type implIAppTokens struct {
	itokens  itokens.ITokens
	appQName appdef.AppQName
}
