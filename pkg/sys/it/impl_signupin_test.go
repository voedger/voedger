/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_SignUpIn(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()
	loginName1 := vit.NextName()
	loginName2 := vit.NextName()

	login1 := vit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)
	login2 := vit.SignUp(loginName2, "pwd2", istructs.AppQName_test1_app1, it.WithClusterID(42))

	prn1 := vit.SignIn(login1)
	prn2 := vit.SignIn(login2)

	require.NotEqual(prn1.Token, prn2.Token)
	require.Equal(istructs.ClusterID(1), istructs.WSID(prn1.ProfileWSID).ClusterID())
	require.Equal(istructs.ClusterID(42), istructs.WSID(prn2.ProfileWSID).ClusterID())
	require.True(prn1.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn2.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn1.ProfileWSID.BaseWSID() != prn2.ProfileWSID.BaseWSID())

	// refresh principal token
	// simulate delay to make the new token be different
	vit.TimeAdd(time.Minute)
	body := fmt.Sprintf(`{"args":{"ExistingPrincipalToken":"%s"},"elements":[{"fields":["NewPrincipalToken"]}]}`, prn1.Token)
	resp := vit.PostProfile(prn1, "q.sys.RefreshPrincipalToken", body)

	refreshedPrincipalToken := resp.SectionRow()[0].(string)
	require.NotEqual(prn1.Token, refreshedPrincipalToken)

	// читать CDoc<Login> не надо. И вообще, в AppWS делать нечего

	var idOfCDocUserProfile int64
	t.Run("check CDoc<sys.UserProfile> at profileWSID at target ap at target cluster", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.UserProfile"},"elements":[{"fields":["sys.ID", "DisplayName"]}]}`
		resp := vit.PostProfile(prn1, "q.sys.Collection", body)
		require.Equal("User Name", resp.SectionRow()[1])
		idOfCDocUserProfile = int64(resp.SectionRow()[0].(float64))
	})

	// checking CDoc<sys.UserProfile> creation is senceless because: in wsid 1 -> 403 foridden + workspace is not initialized, in profile wsid -> singleton violation

	t.Run("modify CDoc<sys.UserProfile> after creation", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"sys.ID": %d,"fields": {"DisplayName":"new name"}}]}`, idOfCDocUserProfile)
		vit.PostProfile(prn1, "c.sys.CUD", body) // nothing to check, just expect no errors here
	})
}

func TestCreateLoginErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	t.Run("wrong url wsid", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"login1","AppName":"test1/app1","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"password"}}`, istructs.SubjectKind_User)
		crc16 := coreutils.CRC16([]byte("login1")) - 1 // simulate crc16 is calculated wrong
		pseudoWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(crc16))
		resp := vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.sys.CreateLogin", body, coreutils.Expect403())
		resp.RequireError(t, "wrong url WSID: 140737488420870 expected, 140737488420869 got")
	})

	login := vit.NextName()
	loginPseudoWSID := coreutils.GetPseudoWSID(login, istructs.MainClusterID)

	t.Run("unknown application", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"my/unknown","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"password"}}`,
			login, istructs.SubjectKind_User, istructs.MainClusterID)
		resp := vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.sys.CreateLogin", body, coreutils.Expect400())
		resp.RequireError(t, "unknown application my/unknown")
	})

	t.Run("wrong application name", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"wrong-AppName","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"different"}}`,
			login, istructs.SubjectKind_User)
		resp := vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.sys.CreateLogin", body, coreutils.Expect400())
		resp.RequireContainsError(t, "failed to parse app qualified name")
	})

	vit.SignUp(login, "1", istructs.AppQName_test1_app1)

	t.Run("create an existing login again", func(t *testing.T) {
		vit.SignUp(login, "1", istructs.AppQName_test1_app1, it.WithReqOpt(coreutils.Expect409()))
	})

	t.Run("subject name constraint violation", func(t *testing.T) {
		// see https://dev.untill.com/projects/#!537026
		wrongLogins := []string{
			"вронг",
			"test@tesT.com",
			"test@test.com ",
			" test@test.com",
			" test@test.com ",
			".test@test.com",
			"test@test.com.",
			".test@test.com.",
			"test@test..com",
			"-test@test.com",
			"test@test.com-",
			"-test@test.com",
			"-test@test.com-",
		}
		for _, wrongLogin := range wrongLogins {
			pseudoWSID := coreutils.GetPseudoWSID(wrongLogin, istructs.MainClusterID)
			body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
				wrongLogin, istructs.AppQName_test1_app1.String(), istructs.SubjectKind_User, istructs.MainClusterID, "1")
			resp := vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.sys.CreateLogin", body, coreutils.Expect400())
			resp.RequireContainsError(t, "incorrect login format")
		}
	})
}

func TestSignInErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	login := vit.NextName()
	pseudoWSID := coreutils.GetPseudoWSID(login, istructs.MainClusterID)

	t.Run("unknown login", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "1","AppName": "%s"},"elements":[{"fields":["PrincipalToken", "WSID", "WSError"]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.IssuePrincipalToken", body, coreutils.Expect401())
	})

	t.Run("wrong WSID", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "1","AppName": "%s"},"elements":[{"fields":[]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, 2, "q.sys.IssuePrincipalToken", body, coreutils.Expect403())
	})

	vit.SignUp(login, "1", istructs.AppQName_test1_app1)

	t.Run("wrong password", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "wrongPass","AppName": "%s"},"elements":[{"fields":[]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.IssuePrincipalToken", body, coreutils.Expect401())
	})
}
