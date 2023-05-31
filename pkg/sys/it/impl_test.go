/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

type greeterRR struct {
	istructs.NullObject
	text string
}

func (e *greeterRR) AsString(name string) string {
	return "hello, " + e.text
}

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	cfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app1, func(vvmCfg *vvm.VVMConfig, vvmAPI vvm.VVMAPI, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {
			cfg.Resources.Add(istructsmem.NewQueryFunction(
				appdef.NewQName(appdef.SysPackage, "Greeter"),
				appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "GreeterParams")).
					AddField("Text", appdef.DataKind_string, true).
					QName(),
				appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "GreeterResult")).
					AddField("Res", appdef.DataKind_string, true).
					QName(),
				func(_ context.Context, _ istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
					text := args.ArgumentObject.AsString("Text")
					var rr = &greeterRR{text: text}
					return callback(rr)
				},
			))

			// need to read cdoc.sys.Subject on auth
			sys.Provide(vvmCfg.TimeFunc, cfg, appDefBuilder, vvmAPI, smtp.Cfg{}, sep, nil)
		}),
	)
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	// отправим POST-запрос
	body := `
	{
		"args": {
		  "Text": "world"
		},
		"elements": [
		  {
			"fields": ["Res"]
		  }
		]
	  }
	`
	ws := vit.DummyWS(istructs.AppQName_test1_app1, 1)
	resp := vit.PostWSSys(ws, "q.sys.Greeter", body)
	require.Equal(`{"sections":[{"type":"","elements":[[[["hello, world"]]]]}]}`, resp.Body)
	resp.Println()
}

func TestAppWSAutoInitialization(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	checkCDocsWSDesc(vit.VVM, require)

	// further calls -> nothing happens, expect not errors
	require.NoError(vvm.BuildAppWorkspaces(vit.VVM, vit.VVMConfig))
	checkCDocsWSDesc(vit.VVM, require)
}

func checkCDocsWSDesc(vvm *vvm.VVM, require *require.Assertions) {
	for _, appQName := range vvm.VVMApps {
		as, err := vvm.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
		for wsNum := 0; istructs.AppWSAmount(wsNum) < as.WSAmount(); wsNum++ {
			as, err := vvm.IAppStructsProvider.AppStructs(appQName)
			require.NoError(err)
			appWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
			existingCDocWSDesc, err := as.Records().GetSingleton(appWSID, sysshared.QNameCDocWorkspaceDescriptor)
			require.NoError(err)
			require.Equal(sysshared.QNameCDocWorkspaceDescriptor, existingCDocWSDesc.QName())
		}
	}
}

func TestAuthorization(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	prn := ws.Owner

	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"%s"}}]}`, it.QNameTestTable)

	t.Run("basic usage", func(t *testing.T) {
		t.Run("Bearer scheme", func(t *testing.T) {
			sys := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
			vit.PostApp(istructs.AppQName_test1_app1, prn.ProfileWSID, "c.sys.CUD", body, coreutils.WithHeaders(coreutils.Authorization, "Bearer "+sys.Token))
		})

		t.Run("Basic scheme", func(t *testing.T) {
			t.Run("token in username only", func(t *testing.T) {
				basicAuthHeader := base64.StdEncoding.EncodeToString([]byte(prn.Token + ":"))
				vit.PostApp(istructs.AppQName_test1_app1, prn.ProfileWSID, "c.sys.CUD", body, coreutils.WithHeaders(coreutils.Authorization, "Basic "+basicAuthHeader))
			})
			t.Run("token in password only", func(t *testing.T) {
				basicAuthHeader := base64.StdEncoding.EncodeToString([]byte(":" + prn.Token))
				vit.PostApp(istructs.AppQName_test1_app1, prn.ProfileWSID, "c.sys.CUD", body, coreutils.WithHeaders(coreutils.Authorization, "Basic "+basicAuthHeader))
			})
			t.Run("token is splitted over username and password", func(t *testing.T) {
				basicAuthHeader := base64.StdEncoding.EncodeToString([]byte(prn.Token[:len(prn.Token)/2] + ":" + prn.Token[len(prn.Token)/2:]))
				vit.PostApp(istructs.AppQName_test1_app1, prn.ProfileWSID, "c.sys.CUD", body, coreutils.WithHeaders(coreutils.Authorization, "Basic "+basicAuthHeader))
			})
		})
	})

	t.Run("401 unauthorized", func(t *testing.T) {
		t.Run("wrong Authorization token", func(t *testing.T) {
			vit.PostProfile(prn, "c.sys.CUD", body, coreutils.WithAuthorizeBy("wrong"), coreutils.Expect401()).Println()
		})

		t.Run("unsupported Authorization header", func(t *testing.T) {
			vit.PostProfile(prn, "c.sys.CUD", body,
				coreutils.WithHeaders(coreutils.Authorization, `whatever w\o Bearer or Basic`), coreutils.Expect401()).Println()
		})

		t.Run("Basic authorization", func(t *testing.T) {
			t.Run("non-base64 header value", func(t *testing.T) {
				vit.PostProfile(prn, "c.sys.CUD", body,
					coreutils.WithHeaders(coreutils.Authorization, `Basic non-base64-value`), coreutils.Expect401()).Println()
			})
			t.Run("no colon between username and password", func(t *testing.T) {
				headerValue := base64.RawStdEncoding.EncodeToString([]byte("some password"))
				vit.PostProfile(prn, "c.sys.CUD", body,
					coreutils.WithHeaders(coreutils.Authorization, "Basic "+headerValue), coreutils.Expect401()).Println()
			})
		})
	})

	t.Run("403 forbidden", func(t *testing.T) {
		t.Run("missing Authorization token", func(t *testing.T) {
			// c.sys.CUD has Owner authorization policy -> need to provide authorization header in PostFunc()
			vit.PostApp(istructs.AppQName_test1_app1, prn.ProfileWSID, "c.sys.CUD", body, coreutils.Expect403()).Println()
		})
	})
}

func TestUtilFuncs(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	t.Run("func Echo", func(t *testing.T) {
		body := `{"args": {"Text": "world"},"elements":[{"fields":["Res"]}]}`
		resp := vit.PostApp(istructs.AppQName_test1_app1, 1, "q.sys.Echo", body)
		require.Equal("world", resp.SectionRow()[0].(string))
		resp.Println()
	})

	t.Run("func GRCount", func(t *testing.T) {
		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := vit.PostApp(istructs.AppQName_test1_app1, 1, "q.sys.GRCount", body)
		resp.Println()
	})
}

func Test400BadRequests(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()
	var err error

	cases := []struct {
		desc     string
		funcName string
		appName  string
	}{
		{desc: "unknown func", funcName: "q.unknown"},
		{desc: "unknown func kind", funcName: "x.test.test"},
		{desc: "wrong resource name", funcName: "a"},
		{desc: "unknown app", funcName: "c.sys.CUD", appName: "un/known"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			appQName := istructs.AppQName_test1_app1
			if len(c.appName) > 0 {
				appQName, err = istructs.ParseAppQName(c.appName)
				require.NoError(t, err)
			}
			vit.PostApp(appQName, 1, c.funcName, "", coreutils.Expect400()).Println()
		})
	}
}

func Test503OnNoQueryProcessorsAvailable(t *testing.T) {
	funcStarted := make(chan interface{})
	okToFinish := make(chan interface{})
	it.MockQryExec = func(input string, callback istructs.ExecQueryCallback) error {
		funcStarted <- nil
		<-okToFinish
		return nil
	}
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
	postDone := sync.WaitGroup{}
	sys := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
	for i := 0; i < int(vit.VVMConfig.NumQueryProcessors); i++ {
		postDone.Add(1)
		go func() {
			defer postDone.Done()
			vit.PostApp(istructs.AppQName_test1_app1, 1, "q.sys.MockQry", body, coreutils.WithAuthorizeBy(sys.Token))
		}()

		<-funcStarted
	}

	vit.PostApp(istructs.AppQName_test1_app1, 1, "q.sys.Echo", body, coreutils.Expect503(), coreutils.WithAuthorizeBy(sys.Token))

	for i := 0; i < int(vit.VVMConfig.NumQueryProcessors); i++ {
		okToFinish <- nil
	}
	postDone.Wait()
}
