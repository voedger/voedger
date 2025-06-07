/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * @author Maxim Geraskin
 *
 */

package itokensjwt

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
)

var SecretKeyTooShortExample = SecretKeyType{
	0x7e, 0x7f, 0x54, 0xfc, 0xbd, 0x1d, 0x6d, 0xe5, 0x6b, 0xc1, 0x01, 0xe7, 0x5c, 0xed, 0x13, 0x5d,
}

type TestPayload_Login struct {
	Login       string
	DisplayName string
	Cluster     int
	SubjectKind istructs.SubjectKindType
	PwdHash     []byte
}

type TestPayload_Principal struct {
	TestPayload_Login
	ProfileWSID istructs.WSID
}

type TestPayload_BLOBUploading struct {
	Workspace istructs.WSID
	BLOB      istructs.RecordID
	MaxSize   int64
}

func TestBasicUsage_ITokens(t *testing.T) {

	require := require.New(t)

	signer := ProvideITokens(SecretKeyExample, testingu.MockTime)

	// Prepare payloads

	principalPayload := TestPayload_Principal{
		TestPayload_Login: TestPayload_Login{
			Login:       "login",
			DisplayName: "displayName",
			Cluster:     32,
			SubjectKind: istructs.SubjectKind_User,
			PwdHash:     []byte{1, 2, 3},
		},
		ProfileWSID: istructs.WSID(123),
	}

	blobberPayload := TestPayload_BLOBUploading{
		Workspace: istructs.WSID(1),
		BLOB:      istructs.RecordID(1),
		MaxSize:   20000,
	}

	// Prepare tokens. NB: payload MUST be passed by reference
	var principalToken, blobToken string
	testAppQName := istructs.AppQName_test1_app1
	testDuration := 1 * time.Minute

	t.Run("Prepare tokens. NB: payload MUST be passed by reference", func(t *testing.T) {
		var err error
		principalToken, err = signer.IssueToken(testAppQName, testDuration, &principalPayload)
		require.NoError(err)
		blobToken, err = signer.IssueToken(testAppQName, testDuration, &blobberPayload)
		require.NoError(err)
	})

	t.Run("Verify principalToken", func(t *testing.T) {
		payload := TestPayload_Principal{}
		gp, err := signer.ValidateToken(principalToken, &payload)
		require.NoError(err)
		require.Equal(testAppQName, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
		require.Equal(principalPayload, payload)
		require.Greater(gp.IssuedAt.Unix(), new(time.Time).Unix())
	})

	t.Run("Verify blobTokenToken", func(t *testing.T) {
		payload := TestPayload_BLOBUploading{}
		gp, err := signer.ValidateToken(blobToken, &payload)
		require.NoError(err)
		require.Equal(testAppQName, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
		require.Greater(gp.IssuedAt.Unix(), new(time.Time).Unix())
		require.Equal(blobberPayload, payload)
	})

	t.Run("Check that principalToken can not be verified for BLOBUploadingPayload", func(t *testing.T) {
		var payload = TestPayload_BLOBUploading{}
		gp, err := signer.ValidateToken(principalToken, &payload)
		require.ErrorIs(err, itokens.ErrInvalidAudience)
		require.Greater(gp.IssuedAt.Unix(), new(time.Time).Unix())
	})

	t.Run("Check that blobToken can not be verified for PrincipalPayload", func(t *testing.T) {
		var payload = TestPayload_Principal{}
		gp, err := signer.ValidateToken(blobToken, &payload)
		require.ErrorIs(err, itokens.ErrInvalidAudience)
		require.Greater(gp.IssuedAt.Unix(), new(time.Time).Unix())
	})

	t.Run("Check expired token", func(t *testing.T) {
		var gp istructs.GenericPayload

		// acquire signer with current time
		expiredToken, err := signer.IssueToken(istructs.AppQName_test1_app1, 1*time.Minute, &principalPayload)
		require.NoError(err)

		// simulate token validity time is passed
		// must get error, because token already expired
		testingu.MockTime.Add(2 * time.Minute)
		var payload = TestPayload_Principal{}
		gp, err = signer.ValidateToken(expiredToken, &payload)
		require.Greater(gp.IssuedAt.Unix(), new(time.Time).Unix())
		require.ErrorIs(err, itokens.ErrTokenExpired)
	})

	t.Run("Check expired token in future", func(t *testing.T) {
		var gp istructs.GenericPayload
		expiredToken, err := signer.IssueToken(istructs.AppQName_test1_app1, testDuration, &principalPayload)
		require.NoError(err)

		// make current time later the expiration moment
		testingu.MockTime.Add(testDuration * 2)
		payload := TestPayload_Principal{}
		gp, err = signer.ValidateToken(expiredToken, &payload)
		require.Greater(gp.IssuedAt.Unix(), new(time.Time).Unix())
		require.ErrorIs(err, itokens.ErrTokenExpired)
	})
}

func TestErrorProcessing(t *testing.T) {
	require := require.New(t)
	signer := TestTokensJWT()

	// Prepare payloads
	principalPayload := TestPayload_Principal{
		TestPayload_Login: TestPayload_Login{
			Login:       "login",
			DisplayName: "displayName",
			Cluster:     32,
			SubjectKind: istructs.SubjectKind_User,
			PwdHash:     []byte{1, 2, 3},
		},
		ProfileWSID: istructs.WSID(123),
	}

	var principalToken string

	t.Run("Issue tokens. Check error when failed unmarshall data.", func(t *testing.T) {
		var err error
		onByteArrayMutate = func(array *[]byte) {
			*array = []byte{0x0A, 0x0D}
		}
		defer testEnvOff()
		principalToken, err = signer.IssueToken(istructs.AppQName_test1_app1, 1*time.Minute, &principalPayload)
		require.ErrorIs(err, itokens.ErrInvalidPayload)
		require.Empty(principalToken)
	})

	t.Run("Issue tokens. Check error on signing token.", func(t *testing.T) {
		var err error
		onSecretKeyMutate = func() interface{} {
			i := make(chan int)
			return i
		}
		defer testEnvOff()
		principalToken, err = signer.IssueToken(istructs.AppQName_test1_app1, 1*time.Minute, &principalPayload)
		require.ErrorIs(err, itokens.ErrSignerError)
		require.Empty(principalToken)
	})

	t.Run("Issue tokens. Send incorrect type of payload. Must receive error", func(t *testing.T) {
		var err error
		wrongTypePayload := make(chan int)
		principalToken, err = signer.IssueToken(istructs.AppQName_test1_app1, 1*time.Minute, wrongTypePayload)
		require.Empty(principalToken)
		require.ErrorIs(err, itokens.ErrInvalidPayload)
	})

	t.Run("Verify incorrect token: full incorrect - bad alg, bad mime etc... Must receive error", func(t *testing.T) {
		var (
			brokenHeader    = "broken_header"
			brokenPayload   = "broken_payload"
			brokenSignature = "broken_signature"
			payload         = TestPayload_Principal{}
		)
		gp, err := signer.ValidateToken(brokenHeader+"."+brokenPayload+"."+brokenSignature, &payload)
		require.Error(err)
		fmt.Printf("%s", time.Time{})
		require.Equal(time.Time{}, gp.IssuedAt)

		gp, err = signer.ValidateToken(brokenHeader+"."+brokenPayload, &payload)
		require.Error(err)
		require.Equal(time.Time{}, gp.IssuedAt)
	})

	t.Run("Verify incorrect token: incorrect algorithm. Must receive error itokens.ErrInvalidToken", func(t *testing.T) {
		var payload = TestPayload_Principal{}
		header := []byte(`
		{
			"alg": "ES256",
			"typ": "JWT"
		}`)
		claims := []byte(`
		{
  			"AppQName": "test1/app1",
  			"Cluster": 32,
  			"DisplayName": "displayName",
  			"Duration": 60000000000,
			"Login": "login",
  			"ProfileWSID": 123,
  			"PwdHash": "AQID",
  			"SubjectKind": 1,
  			"aud": "itokensjwt.TestPayload_Principal",
  			"exp": 1646987911,
  			"iat": 1646987851
		}`)

		token := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims) + "." + "x86NBi8jOX47MvQj0o9IVzdetGF6VUi_f0o0o-DeBcY"
		gp, err := signer.ValidateToken(token, &payload)
		require.ErrorIs(err, itokens.ErrInvalidToken)
		require.Equal(time.Time{}, gp.IssuedAt)
	})

	t.Run("Verify tokens. Check error when unmarshall token claims.", func(t *testing.T) {
		var (
			payload = TestPayload_Principal{}
			err     error
		)
		principalToken, err = signer.IssueToken(istructs.AppQName_test1_app1, 1*time.Minute, &principalPayload)
		require.NoError(err)
		onByteArrayMutate = func(array *[]byte) {
			*array = []byte{0x0A, 0x0D}
		}
		defer testEnvOff()
		_, err = signer.ValidateToken(principalToken, &payload)
		require.Error(err)
	})

	t.Run("Verify tokens. Check error on decode base64 part. Must get CorruptInputError", func(t *testing.T) {
		var (
			payload = TestPayload_Principal{}
			err     error
		)
		principalToken, err = signer.IssueToken(istructs.AppQName_test1_app1, 1*time.Minute, &principalPayload)
		require.NoError(err)
		onTokenPartsMutate = func(str []string) string {
			return "!!!!!"
		}
		defer testEnvOff()
		_, err = signer.ValidateToken(principalToken, &payload)
		require.Error(err)
	})

	t.Run("Verify tokens. Check parts count in token. Must get itokens.ErrInvalidPayload", func(t *testing.T) {
		var (
			payload = TestPayload_Principal{}
			err     error
		)
		principalToken, err = signer.IssueToken(istructs.AppQName_test1_app1, 1*time.Minute, &principalPayload)
		require.NoError(err)
		onTokenArrayPartsMutate = func() []string {
			return []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
				"eyJBcHBRTmFtZSI6InRlc3QxL2FwcDEiLCJDbHVzdGVyIjozMiwiRGlzcGxheU5hbWUiOiJkaXNwbGF5TmFtZSIsIkR1cmF0aW9uIjo2MDAwMDAwMDAwMCwiTG9naW4iOiJsb2dpbiIsIlByb2ZpbGVXU0lEIjoxMjMsIlB3ZEhhc2giOiJBUUlEIiwiU3ViamVjdEtpbmQiOjEsImF1ZCI6Iml0b2tlbnNqd3QuVGVzdFBheWxvYWRfUHJpbmNpcGFsIiwiZXhwIjoxNjQ2OTg3OTExLCJpYXQiOjE2NDY5ODc4NTF9",
			}
		}
		defer testEnvOff()
		_, err = signer.ValidateToken(principalToken, &payload)
		require.Error(err)
	})

}

// Try to create signer with TOO SHORT Secret Key. We must panic.
func TestSecretKeyTooShort(t *testing.T) {
	require.Panics(t, func() {
		_ = ProvideITokens(SecretKeyTooShortExample, testingu.MockTime)
	})
}

func TestHashFunc(t *testing.T) {
	var (
		// in hex "Test for hmac256"
		testMsg = []byte{0x54, 0x65, 0x73, 0x74, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x68, 0x6d, 0x61, 0x63, 0x32, 0x35, 0x36}
		hashMsg = []byte{0x2b, 0x00, 0xd0, 0x3a, 0x24, 0xa9, 0x1f, 0x1e, 0xf1, 0x06, 0x24, 0x5a, 0x10, 0x3f, 0x20, 0x10,
			0xe7, 0x82, 0xae, 0xdd, 0x2e, 0x2c, 0x55, 0x01, 0xf1, 0x34, 0x2b, 0x00, 0x58, 0xce, 0x89, 0xed}
	)
	b := make([]byte, 32)
	require := require.New(t)
	signer := TestTokensJWT()
	hash := signer.CryptoHash256(testMsg)
	require.NotNil(hash)
	copy(b, hash[:])
	require.Equal(hashMsg, b)
}

func testEnvOff() {
	onByteArrayMutate = nil
	onSecretKeyMutate = nil
	onTokenPartsMutate = nil
	onTokenArrayPartsMutate = nil
}
