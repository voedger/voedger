/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package payloads

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

var (
	testApp      = istructs.AppQName_test1_app1
	testDuration = time.Minute
)

func TestBasicUsage_PrincipalPayload(t *testing.T) {

	require := require.New(t)

	signer := itokensjwt.TestTokensJWT()

	srcPayload := PrincipalPayload{
		Login:       "login",
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: istructs.WSID(10),
	}

	var token string
	var err error

	t.Run("Prepare token", func(t *testing.T) {
		token, err = signer.IssueToken(testApp, testDuration, &srcPayload)
		log.Printf("%+v", srcPayload)
		require.NoError(err)
	})

	t.Run("Verify token", func(t *testing.T) {
		payload := PrincipalPayload{}
		gp, err := signer.ValidateToken(token, &payload)
		require.NoError(err)
		require.Equal(srcPayload, payload)
		require.Greater(gp.IssuedAt.Unix(), int64(0))
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
	})

}

func TestBasicUsage_BLOBUploadingPayload(t *testing.T) {

	require := require.New(t)
	signer := itokensjwt.TestTokensJWT()

	srcPayload := BLOBUploadingPayload{
		Workspace: istructs.WSID(1),
		BLOB:      istructs.RecordID(1),
		MaxSize:   20000,
	}

	var token string
	var err error

	t.Run("Prepare token", func(t *testing.T) {
		token, err = signer.IssueToken(testApp, testDuration, &srcPayload)
		log.Printf("%+v", srcPayload)
		require.NoError(err)
	})

	t.Run("Verify token", func(t *testing.T) {
		payload := BLOBUploadingPayload{}
		gp, err := signer.ValidateToken(token, &payload)
		require.NoError(err)
		require.Equal(srcPayload, payload)
		require.Greater(gp.IssuedAt.Unix(), int64(0))
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
	})
}

func TestBasicUsage_VerifiedValue(t *testing.T) {

	require := require.New(t)
	signer := itokensjwt.TestTokensJWT()
	testQName := appdef.NewQName("test", "entity")

	token := ""
	var err error

	t.Run("Issue token", func(t *testing.T) {
		payload := VerifiedValuePayload{
			VerificationKind: appdef.VerificationKind_EMail,
			WSID:             43,
			Entity:           testQName,
			Field:            "testName",
			Value:            42,
		}
		token, err = signer.IssueToken(testApp, testDuration, &payload)
		require.NoError(err)
	})

	t.Run("Verify token", func(t *testing.T) {
		payload := VerifiedValuePayload{}
		gp, err := signer.ValidateToken(token, &payload)
		require.NoError(err)
		require.Equal(appdef.VerificationKind_EMail, payload.VerificationKind)
		require.Equal(testQName, payload.Entity)
		require.Equal("testName", payload.Field)
		require.Equal(json.Number("42"), payload.Value)
		require.Greater(gp.IssuedAt.Unix(), int64(0))
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
		require.Equal(istructs.WSID(43), payload.WSID)
	})
}

func TestBasicUsage_IAppTokens(t *testing.T) {
	require := require.New(t)
	tokens := itokensjwt.TestTokensJWT()
	atf := ProvideIAppTokensFactory(tokens)
	at := atf.New(testApp)

	token := ""
	var err error

	t.Run("Issue token", func(t *testing.T) {
		srcPayload := PrincipalPayload{
			Login:       "login",
			SubjectKind: istructs.SubjectKind_User,
			ProfileWSID: istructs.WSID(10),
		}
		token, err = at.IssueToken(testDuration, &srcPayload)
		require.NoError(err)
	})

	t.Run("Validate token", func(t *testing.T) {
		payload := PrincipalPayload{}
		gp, err := at.ValidateToken(token, &payload)
		require.NoError(err)
		require.Greater(gp.IssuedAt.Unix(), int64(0))
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
	})

	t.Run("Basic validation error", func(t *testing.T) {
		testingu.MockTime.Add(testDuration * 2)
		defer func() { testingu.MockTime.Add(-testDuration * 2) }()
		payload := PrincipalPayload{}
		_, err := at.ValidateToken(token, &payload)
		require.ErrorIs(err, itokens.ErrTokenExpired)
	})

	t.Run("Error on validate a token issued for an another app", func(t *testing.T) {
		tokens := itokensjwt.TestTokensJWT()
		atf := ProvideIAppTokensFactory(tokens)
		at := atf.New(istructs.AppQName_test2_app1)
		payload := PrincipalPayload{}
		_, err := at.ValidateToken(token, &payload)
		require.Equal(ErrTokenIssuedForAnotherApp, err)
	})
}
