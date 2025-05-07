/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
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

// [~server.n10n.heartbeats/it.Heartbeat30~impl]
func TestBasicUsage_Heartbeat30(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1, // it does NOT matter
		Projection: in10n.QNameHeartbeat30,       // it DOES matter
		WS:         ws.WSID,                      // it does NOT matter
	}

	endCh := make(chan struct{})
	var wg sync.WaitGroup

	// Start a goroutine to simulate the passage of time
	{
		wg.Add(1)
		go func() {

			defer func() {
				wg.Done()
				close(endCh)
			}()

			for {
				select {
				case <-endCh:
					logger.Info("TestBasicUsage_Heartbeat30: endCh")
					return
				case <-time.After(100 * time.Millisecond):
					logger.Info("TestBasicUsage_Heartbeat30: MockTime.Add()")
					testingu.MockTime.Add(in10n.Heartbeat30Duration)
				}
			}

		}()
	}

	logger.Info("TestBasicUsage_Heartbeat30: before SubscribeForN10nUnsubscribe, key:", testProjectionKey)
	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(testProjectionKey)

	select {
	case <-offsetsChan:
		logger.Info("TestBasicUsage_Heartbeat30: received heartbeat notification")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for heartbeat notification")
	}

	unsubscribe()
	endCh <- struct{}{}
	wg.Wait()
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
