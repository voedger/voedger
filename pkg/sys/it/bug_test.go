/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

type rr struct {
	istructs.NullObject
	res string
}

func (r *rr) AsString(string) string {
	return r.res
}

func TestBug_QueryProcessorMustStopOnClientDisconnect(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)
	goOn := make(chan interface{})
	it.MockQryExec = func(input string, callback istructs.ExecQueryCallback) (err error) {
		rr := &rr{res: input}
		require.NoError(callback(rr))
		<-goOn // ждем, пока http клиент примет первый элемент и отключится
		// теперь ждем ошибку context.Cancelled. Она выйдет не сразу, т.к. в queryprocessor работает асинхронный конвейер
		for err == nil {
			err = callback(rr)
		}
		require.Equal(context.Canceled, err)
		defer func() { goOn <- nil }() // отсигналим, что поймали ошибку context.Cancelled
		return err
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// отправим POST-запрос
	body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
	ws := vit.DummyWS(istructs.AppQName_test1_app1)
	vit.PostWSSys(ws, "q.app1pkg.MockQry", body, coreutils.WithResponseHandler(func(httpResp *http.Response) {
		// прочтем первую часть ответа (сервер не отдаст вторую, пока в goOn не запишем чего-нибудь)
		entireResp := []byte{}
		var err error
		n := 0
		for string(entireResp) != `{"sections":[{"type":"","elements":[[[["world"]]]` {
			if n == 0 && err == io.EOF {
				t.Fatal()
			}
			buf := make([]byte, 512)
			n, err = httpResp.Body.Read(buf)
			entireResp = append(entireResp, buf[:n]...)
		}

		// порвем соединение в середине обработки запроса
		httpResp.Request.Body.Close()
		httpResp.Body.Close()
		goOn <- nil // функция начнет передавать вторую часть, но это не получится, т.к. request context закрыт
	}))

	<-goOn // подождем, пока ошибки проверятся
	// ожидаем, что никаких посторонних ошибок нет: ничего не повисло, queryprocessor отдал управление, роутер не пытается писать в закрытую коннекцию и т.п.
}

func Test409OnRepeatedlyUsedRawIDsInResultCUDs(t *testing.T) {
	vitCfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app2, func(apis apps.APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) apps.AppPackages {

			sysPackageFS := sys.Provide(cfg, appDefBuilder, smtp.Cfg{}, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
				apis.NumCommandProcessors, nil, apis.IAppStorageProvider)

			qNameDoc1 := appdef.NewQName("app2pkg", "doc1")
			// appDefBuilder.AddCDoc(cdocQName)

			cmdQName := appdef.NewQName("app2pkg", "testCmd")
			cmd2CUDs := istructsmem.NewCommandFunction(cmdQName,
				func(args istructs.ExecCommandArgs) (err error) {
					// 2 раза используем один и тот же rawID -> 500 internal server error
					kb, err := args.State.KeyBuilder(state.Record, qNameDoc1)
					if err != nil {
						return
					}
					sv, err := args.Intents.NewValue(kb)
					if err != nil {
						return
					}
					sv.PutRecordID(appdef.SystemField_ID, 1)

					kb, err = args.State.KeyBuilder(state.Record, qNameDoc1)
					if err != nil {
						return
					}
					sv, err = args.Intents.NewValue(kb)
					if err != nil {
						return
					}
					sv.PutRecordID(appdef.SystemField_ID, 1)
					return nil
				},
			)
			cfg.Resources.Add(cmd2CUDs)
			appPackageFS := parser.PackageFS{
				QualifiedPackageName: "github.com/voedger/voedger/pkg/vit/app2pkg",
				FS:                   it.SchemaTestApp2FS,
			}
			return apps.AppPackages{
				AppQName: istructs.AppQName_test1_app2,
				Packages: []parser.PackageFS{sysPackageFS, appPackageFS},
			}
		}, it.WithUserLogin("login", "1")),
	)
	vit := it.NewVIT(t, &vitCfg)
	defer vit.TearDown()

	prn := vit.GetPrincipal(istructs.AppQName_test1_app2, "login")
	resp := vit.PostProfile(prn, "c.app2pkg.testCmd", "{}", coreutils.Expect409())
	resp.Println()
}
