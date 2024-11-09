/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_SignUpIn(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	loginName1 := vit.NextName()
	loginName2 := vit.NextName()

	login1 := vit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)
	login2 := vit.SignUp(loginName2, "pwd2", istructs.AppQName_test1_app1) // now wrong to create a login in a different CLusterID because it is unknown how to init AppWorkspace there

	prn1 := vit.SignIn(login1)
	prn2 := vit.SignIn(login2)

	require.NotEqual(prn1.Token, prn2.Token)
	require.Equal(istructs.ClusterID(1), prn1.ProfileWSID.ClusterID())
	require.Equal(istructs.ClusterID(1), prn2.ProfileWSID.ClusterID())
	require.True(prn1.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn2.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn1.ProfileWSID.BaseWSID() != prn2.ProfileWSID.BaseWSID())

	// refresh principal token
	// simulate delay to make the new token be different
	vit.TimeAdd(time.Minute)
	body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
	resp := vit.PostProfile(prn1, "q.sys.RefreshPrincipalToken", body)

	refreshedPrincipalToken := resp.SectionRow()[0].(string)
	require.NotEqual(prn1.Token, refreshedPrincipalToken)

	// not need to read CDoc<Login>. Nothing to do in AppWS at all.

	var idOfCDocUserProfile int64
	t.Run("check CDoc<sys.UserProfile> at profileWSID at target app at target cluster", func(t *testing.T) {
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

	t.Run("check token default TTL", func(t *testing.T) {
		var p payloads.PrincipalPayload
		gp, err := vit.ITokens.ValidateToken(prn1.Token, &p)
		require.NoError(err)
		require.Equal(authnz.DefaultPrincipalTokenExpiration, gp.Duration)
	})
}

func TestTTL(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	t.Run("default TTL", func(t *testing.T) {
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		var p payloads.PrincipalPayload
		gp, err := vit.ITokens.ValidateToken(prn.Token, &p)
		require.NoError(err)
		require.Equal(authnz.DefaultPrincipalTokenExpiration, gp.Duration)
	})

	t.Run("custom TTL", func(t *testing.T) {
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "%s","AppName": "%s", "TTLHours":15},"elements":[{"fields":["PrincipalToken"]}]}`,
			prn.Name, prn.Pwd, prn.AppQName.String())
		resp := vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body)
		token := resp.SectionRow()[0].(string)
		var p payloads.PrincipalPayload
		gp, err := vit.ITokens.ValidateToken(token, &p)
		require.NoError(err)
		require.Equal(15*time.Hour, gp.Duration)
	})
}

func TestCreateLoginErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	t.Run("wrong AppWSID", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"login1","AppName":"test1/app1","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"password"}}`, istructs.SubjectKind_User)
		crc16 := coreutils.CRC16([]byte("login1")) - 1 // simulate crc16 is calculated wrong
		pseudoWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(crc16))
		resp := vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.registry.CreateLogin", body, coreutils.Expect403())
		resp.RequireError(t, "wrong AppWSID: 140737488420870 expected, 140737488420869 got")
	})

	login := vit.NextName()
	loginPseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

	t.Run("unknown application", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"my/unknown","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"password"}}`,
			login, istructs.SubjectKind_User, istructs.CurrentClusterID())
		vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.registry.CreateLogin", body, coreutils.Expect400("my/unknown is not found"))
	})

	t.Run("wrong application name", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"wrong-AppName","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"different"}}`,
			login, istructs.SubjectKind_User)
		resp := vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.registry.CreateLogin", body, coreutils.Expect400())
		resp.RequireContainsError(t, "failed to parse app qualified name")
	})

	newLogin := vit.SignUp(login, "1", istructs.AppQName_test1_app1)
	// wait for acomplishing the profile init
	vit.SignIn(newLogin)

	t.Run("create an existing login again", func(t *testing.T) {
		vit.SignUp(login, "1", istructs.AppQName_test1_app1, it.WithReqOpt(coreutils.Expect409()))
	})

	t.Run("subject name constraint violation", func(t *testing.T) {
		// see https://dev.untill.com/projects/#!537026
		wrongLogins := []string{
			"哇",
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
			"sys.test@test.com",
		}
		for _, wrongLogin := range wrongLogins {
			pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, wrongLogin, istructs.CurrentClusterID())
			body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
				wrongLogin, istructs.AppQName_test1_app1.String(), istructs.SubjectKind_User, istructs.CurrentClusterID(), "1")
			resp := vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.registry.CreateLogin", body, coreutils.Expect400())
			resp.RequireContainsError(t, "incorrect login format")
		}
	})
}

func TestSignInErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	login := vit.NextName()
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

	t.Run("unknown login", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "1","AppName": "%s"},"elements":[{"fields":["PrincipalToken", "WSID", "WSError"]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.IssuePrincipalToken", body, coreutils.Expect401()).Println()
	})

	newLogin := vit.SignUp(login, "1", istructs.AppQName_test1_app1)
	// wait for acomplishing the profile init
	vit.SignIn(newLogin)

	t.Run("wrong password", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "wrongPass","AppName": "%s"},"elements":[{"fields":[]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.IssuePrincipalToken", body, coreutils.Expect401()).Println()
	})

	t.Run("wrong TTL", func(t *testing.T) {
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "%s","AppName": "%s", "TTLHours":1000},"elements":[{"fields":["PrincipalToken"]}]}`,
			prn.Name, prn.Pwd, prn.AppQName.String())
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body,
			coreutils.Expect400("max token TTL hours is 168 hours"))
	})
}

func TestDeviceProfile(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	loginName := vit.NextName()
	deviceLogin := vit.SignUpDevice(loginName, "123", istructs.AppQName_test1_app2)
	devicePrn := vit.SignIn(deviceLogin)
	as, err := vit.BuiltIn(istructs.AppQName_test1_app2)
	require.NoError(err)
	devicePrnPayload := payloads.PrincipalPayload{}
	_, err = as.AppTokens().ValidateToken(devicePrn.Token, &devicePrnPayload)
	require.NoError(err)
	require.Equal(istructs.SubjectKind_Device, devicePrnPayload.SubjectKind)

	t.Run("exec a simple operation in the device profile", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostProfile(devicePrn, "q.sys.Collection", body)
	})

	t.Run("refresh the device principal token", func(t *testing.T) {
		// simulate delay to make the new token be different
		vit.TimeAdd(time.Minute)
		body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
		resp := vit.PostProfile(devicePrn, "q.sys.RefreshPrincipalToken", body)
		require.NotEqual(devicePrn.Token, resp.SectionRow()[0].(string))
	})

}
