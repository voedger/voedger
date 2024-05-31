/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 */

package itokens

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type ITokens interface {

	// Payload must be sent by reference
	// All payload fields go to token payload
	// Audience = payload.Type.package + payload.Type.Name
	IssueToken(app appdef.AppQName, duration time.Duration, pointerToPayload interface{}) (token string, err error)

	// Payload must be sent by reference
	// payload MUST be a pointer to the struct of the type used in IssueToken (checked using Audience)
	// Token is verified and its data is copied to *pointerToPayload
	// ErrTokenExpired, ErrInvalidToken, ErrInvalidAudience might be returned
	ValidateToken(token string, pointerToPayload interface{}) (gp istructs.GenericPayload, err error)

	// CryptoHash256 must be a cryptographic hash function which produces a 256-bits hash
	// Ref. https://en.wikipedia.org/wiki/Cryptographic_hash_function
	// Ref. https://pkg.go.dev/crypto/sha256
	CryptoHash256(data []byte) (hash [32]byte)
}
