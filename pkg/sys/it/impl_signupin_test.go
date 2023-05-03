/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_SignUpIn(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()
	loginName1 := hit.NextName()
	loginName2 := hit.NextName()

	login1 := hit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)
	login2 := hit.SignUp(loginName2, "pwd2", istructs.AppQName_test1_app1, it.WithClusterID(42))

	prn1 := hit.SignIn(login1)
	prn2 := hit.SignIn(login2)

	require.NotEqual(prn1.Token, prn2.Token)
	require.Equal(istructs.ClusterID(1), istructs.WSID(prn1.ProfileWSID).ClusterID())
	require.Equal(istructs.ClusterID(42), istructs.WSID(prn2.ProfileWSID).ClusterID())
	require.True(prn1.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn2.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn1.ProfileWSID.BaseWSID() != prn2.ProfileWSID.BaseWSID())

	// refresh principal token
	// simulate delay to make the new token be different
	hit.TimeAdd(time.Minute)
	body := fmt.Sprintf(`{"args":{"ExistingPrincipalToken":"%s"},"elements":[{"fields":["NewPrincipalToken"]}]}`, prn1.Token)
	resp := hit.PostProfile(prn1, "q.sys.RefreshPrincipalToken", body)

	refreshedPrincipalToken := resp.SectionRow()[0].(string)
	require.NotEqual(prn1.Token, refreshedPrincipalToken)

	// читать CDoc<Login> не надо. И вообще, в AppWS делать нечего

	var idOfCDocUserProfile int64
	t.Run("check CDoc<sys.UserProfile> at profileWSID at target ap at target cluster", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.UserProfile"},"elements":[{"fields":["sys.ID", "DisplayName"]}]}`
		resp := hit.PostProfile(prn1, "q.sys.Collection", body)
		require.Equal("User Name", resp.SectionRow()[1])
		idOfCDocUserProfile = int64(resp.SectionRow()[0].(float64))
	})

	// checking CDoc<sys.UserProfile> creation is senceless because: in wsid 1 -> 403 foridden + workspace is not initialized, in profile wsid -> singleton violation

	t.Run("modify CDoc<sys.UserProfile> after creation", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"sys.ID": %d,"fields": {"DisplayName":"new name"}}]}`, idOfCDocUserProfile)
		hit.PostProfile(prn1, "c.sys.CUD", body) // nothing to check, just expect no errors here
	})
}

func TestCreateLoginErrors(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	t.Run("wrong url wsid", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"login1","AppName":"test1/app1","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"password"}}`, istructs.SubjectKind_User)
		crc16 := utils.CRC16([]byte("login1")) - 1 // simulate crc16 is calculated wrong
		pseudoWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(crc16))
		resp := hit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.sys.CreateLogin", body, utils.Expect403())
		resp.RequireError(t, "wrong url WSID: 140737488420870 expected, 140737488420869 got")
	})

	login := hit.NextName()
	loginPseudoWSID := utils.GetPseudoWSID(login, istructs.MainClusterID)

	t.Run("unknown application", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"my/unknown","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"password"}}`,
			login, istructs.SubjectKind_User, istructs.MainClusterID)
		resp := hit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.sys.CreateLogin", body, utils.Expect400())
		resp.RequireError(t, "unknown application my/unknown")
	})

	t.Run("wrong application name", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"wrong-AppName","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"different"}}`,
			login, istructs.SubjectKind_User)
		resp := hit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.sys.CreateLogin", body, utils.Expect400())
		resp.RequireContainsError(t, "failed to parse app qualified name")
	})

	hit.SignUp(login, "1", istructs.AppQName_test1_app1)

	t.Run("create an existing login again", func(t *testing.T) {
		hit.SignUp(login, "1", istructs.AppQName_test1_app1, it.WithReqOpt(utils.Expect409()))
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
			pseudoWSID := utils.GetPseudoWSID(wrongLogin, istructs.MainClusterID)
			body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
				wrongLogin, istructs.AppQName_test1_app1.String(), istructs.SubjectKind_User, istructs.MainClusterID, "1")
			resp := hit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.sys.CreateLogin", body, utils.Expect400())
			resp.RequireContainsError(t, "incorrect login format")
		}
	})
}

func TestSignInErrors(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	login := hit.NextName()
	pseudoWSID := utils.GetPseudoWSID(login, istructs.MainClusterID)

	t.Run("unknown login", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "1","AppName": "%s"},"elements":[{"fields":["PrincipalToken", "WSID", "WSError"]}]}`,
			login, istructs.AppQName_test1_app1.String())
		hit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.IssuePrincipalToken", body, utils.Expect401())
	})

	t.Run("wrong WSID", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "1","AppName": "%s"},"elements":[{"fields":[]}]}`,
			login, istructs.AppQName_test1_app1.String())
		hit.PostApp(istructs.AppQName_sys_registry, 2, "q.sys.IssuePrincipalToken", body, utils.Expect403())
	})

	hit.SignUp(login, "1", istructs.AppQName_test1_app1)

	t.Run("wrong password", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "wrongPass","AppName": "%s"},"elements":[{"fields":[]}]}`,
			login, istructs.AppQName_test1_app1.String())
		hit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.IssuePrincipalToken", body, utils.Expect401())
	})
}
