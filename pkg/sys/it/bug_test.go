/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

type rr struct {
	istructs.NullObject
	res string
}

func (r *rr) AsString(string) string {
	return r.res
}

func TestBug_QueryProcessorMustStopOnClientDisconnect(t *testing.T) {
	require := require.New(t)
	goOn := make(chan interface{})
	it.MockQryExec = func(input string, callback istructs.ExecQueryCallback) (err error) {
		rr := &rr{res: input}
		require.Nil(callback(rr))
		<-goOn // ждем, пока http клиент примет первый элемент и отключится
		// теперь ждем ошибку context.Cancelled. Она выйдет не сразу, т.к. в queryprocessor работает асинхронный конвейер
		for err == nil {
			err = callback(rr)
		}
		require.Equal(context.Canceled, err)
		defer func() { goOn <- nil }() // отсигналим, что поймали ошибку context.Cancelled
		return err
	}
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	// отправим POST-запрос
	body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
	ws := hit.DummyWS(istructs.AppQName_test1_app1)
	hit.PostWSSys(ws, "q.sys.MockQry", body, coreutils.WithResponseHandler(func(httpResp *http.Response) {
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
	hitCfg := it.NewOwnHITConfig(
		it.WithApp(istructs.AppQName_test1_app1, func(hvmCfg *vvm.HVMConfig, hvmAPI vvm.HVMAPI, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {

			sys.Provide(hvmCfg.TimeFunc, cfg, adf, hvmAPI, smtp.Cfg{}, sep)

			cdocQName := appdef.NewQName("test", "cdoc")
			adf.AddStruct(cdocQName, appdef.DefKind_CDoc)

			cmdQName := appdef.NewQName(appdef.SysPackage, "testCmd")
			cmd2CUDs := istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName,
				func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
					// 2 раза используем один и тот же rawID -> 500 internal server error
					kb, err := args.State.KeyBuilder(state.RecordsStorage, cdocQName)
					if err != nil {
						return
					}
					sv, err := args.Intents.NewValue(kb)
					if err != nil {
						return
					}
					sv.PutRecordID(appdef.SystemField_ID, 1)

					kb, err = args.State.KeyBuilder(state.RecordsStorage, cdocQName)
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
		}, it.WithUserLogin("login", "1")),
	)
	hit := it.NewHIT(t, &hitCfg)
	defer hit.TearDown()

	prn := hit.GetPrincipal(istructs.AppQName_test1_app1, "login")
	resp := hit.PostProfile(prn, "c.sys.testCmd", "{}", utils.Expect409())
	resp.Println()
}
