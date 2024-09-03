/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package itokensjwt

import (
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/istructs"
)

func Benchmark_IssueTtoken(b *testing.B) {

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

	testAppQName := istructs.AppQName_test1_app1
	testDuration := 1 * time.Minute

	for i := 0; i < b.N; i++ {
		_, err := signer.IssueToken(testAppQName, testDuration, &principalPayload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_VerifyToken(b *testing.B) {
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

	testAppQName := istructs.AppQName_test1_app1
	testDuration := 1 * time.Minute

	principalToken, err := signer.IssueToken(testAppQName, testDuration, &principalPayload)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		payload := TestPayload_Principal{}
		_, err := signer.ValidateToken(principalToken, &payload)
		if err != nil {
			b.Fatal(err)
		}
	}

}
