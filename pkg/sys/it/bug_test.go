/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
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
	it.MockQryExec = func(input string, _ istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
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
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	vit.PostWS(ws, "q.app1pkg.MockQry", body, coreutils.WithResponseHandler(func(httpResp *http.Response) {
		// прочтем первую часть ответа (сервер не отдаст вторую, пока в goOn не запишем чего-нибудь)
		entireResp := []byte{}
		var err error
		n := 0
		for string(entireResp) != `{"sections":[{"type":"","elements":[[[["world"]]]` {
			if n == 0 && errors.Is(err, io.EOF) {
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

func Test409OnRepeatedlyUsedRawIDsInResultCUDs_(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	it.MockCmdExec = func(_ string, args istructs.ExecCommandArgs) error {
		// 2 раза используем один и тот же rawID -> 500 internal server error
		kb, err := args.State.KeyBuilder(sys.Storage_Record, it.QNameApp1_CDocCategory)
		if err != nil {
			return err
		}
		sv, err := args.Intents.NewValue(kb)
		if err != nil {
			return err
		}
		sv.PutRecordID(appdef.SystemField_ID, 1)

		kb, err = args.State.KeyBuilder(sys.Storage_Record, it.QNameApp1_CDocCategory)
		if err != nil {
			return err
		}
		sv, err = args.Intents.NewValue(kb)
		if err != nil {
			return err
		}
		sv.PutRecordID(appdef.SystemField_ID, 1)
		return nil
	}
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	vit.PostWS(ws, "c.app1pkg.MockCmd", `{"args":{"Input":"Str"}}`, coreutils.Expect409()).Println()
}
