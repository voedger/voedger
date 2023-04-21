/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 */

package payloads

import (
	"github.com/voedger/voedger/pkg/istructs"
	itokens "github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/schemas"
)

// Principal can be referenced by WSID
// Owner is a record with {WSID, IDOfOwner} key
// isAPIToken -> principals will be built by Roles only in authenticator
type PrincipalPayload struct {
	Login       string
	SubjectKind istructs.SubjectKindType
	ProfileWSID istructs.WSID
	Roles       []RoleType
	IsAPIToken  bool
}

type RoleType struct {
	WSID istructs.WSID
	// E.g. air.LinkedDevice
	QName schemas.QName
}

type BLOBUploadingPayload struct {
	Workspace istructs.WSID
	BLOB      istructs.RecordID
	MaxSize   int64
}

type VerificationKindType uint8

const (
	VerificationKind_EMail VerificationKindType = iota
	VerificationKind_Phone
	VerificationKind_FakeLast
)

type VerifiedValuePayload struct {
	VerificationKind VerificationKindType
	WSID             istructs.WSID
	ID               istructs.RecordID
	Entity           schemas.QName
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
	appQName istructs.AppQName
}
