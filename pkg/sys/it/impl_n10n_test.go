/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_n10n(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("paa", "price"),
		WS:         ws.WSID,
	}

	offsetsChan, unsubscribe, err := vit.N10NSubscribe(testProjectionKey)
	require.NoError(err)

	done := make(chan interface{})
	go func() {
		defer close(done)
		for offset := range offsetsChan {
			require.Equal(istructs.Offset(13), offset)
		}
	}()

	// вызовем тестовый метод update для обновления проекции
	vit.N10NUpdate(testProjectionKey, 13)

	// отпишемся, чтобы закрылся канал offsetsChan
	unsubscribe()

	<-done // подождем чтения события и закрытия каналс offsestChan
}
