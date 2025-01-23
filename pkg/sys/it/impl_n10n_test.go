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
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("app1pkg", "CategoryIdx"),
		WS:         ws.WSID,
	}

	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(testProjectionKey)

	// force projection update
	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	resultOffsetOfCUD := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset
	require.EqualValues(t, resultOffsetOfCUD, <-offsetsChan)
	unsubscribe()

	_, offsetsChanOpened := <-offsetsChan
	require.False(t, offsetsChanOpened)
}

func TestSynthetic(t *testing.T) {
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

	// call a test method that updates the projection
	vit.N10NUpdate(testProjectionKey, 13)

	// unsubscribe to force offsetsChan to close
	unsubscribe()

	<-done // wait for event read and offsestChan close
}
