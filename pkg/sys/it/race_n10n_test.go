/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
	it "github.com/voedger/voedger/pkg/vit"
)

// Test_Race_n10nCHS: Create channel and wait for subscribe
func Test_Race_n10nCHS(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	wg := &sync.WaitGroup{}
	cnt := 100
	unsubscribes := []func(){}
	offsetsChans := []federation.OffsetsChan{}
	for wsid := istructs.WSID(0); wsid < istructs.WSID(cnt); wsid++ {
		wg.Add(1)
		go func(wsid istructs.WSID) {
			defer wg.Done()
			offsetsChan, unsubscribe, err := vit.N10NSubscribe(in10n.ProjectionKey{
				App:        istructs.NewAppQName("untill", "Application"),
				Projection: appdef.NewQName("paa", "price"),
				WS:         wsid,
			})
			require.NoError(t, err)
			unsubscribes = append(unsubscribes, unsubscribe)
			offsetsChans = append(offsetsChans, offsetsChan)
		}(wsid)
	}
	wg.Wait()
	for _, unsubscribe := range unsubscribes {
		unsubscribe()
	}
	for _, offsetsChan := range offsetsChans {
		for range offsetsChan {
		}
	}
}

// Test_Race_n10nCHSU: Create channel,  read event, send update
func Test_Race_n10nCHSU(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	wg := sync.WaitGroup{}
	cnt := 10
	unsubscribes := []func(){}
	offsetsChans := []federation.OffsetsChan{}
	for wsid := istructs.WSID(0); wsid < istructs.WSID(cnt); wsid++ {
		wg.Add(1)
		go func(wsid istructs.WSID) {
			defer wg.Done()

			// create chan and subscribe
			projectionKey := in10n.ProjectionKey{
				App:        istructs.AppQName_test1_app1,
				Projection: appdef.NewQName("paa", "price"),
				WS:         wsid,
			}
			offsetsChan, unsubsribe, err := vit.N10NSubscribe(projectionKey)
			require.NoError(t, err)
			unsubscribes = append(unsubscribes, unsubsribe)
			offsetsChans = append(offsetsChans, offsetsChan)

			// upate
			require.NoError(t, vit.N10NUpdate(projectionKey, 13))
		}(wsid)
	}
	wg.Wait()
	wg = sync.WaitGroup{}
	for _, offsetsChan := range offsetsChans {
		wg.Add(1)
		go func(offsetsChan federation.OffsetsChan) {
			defer wg.Done()
			for offset := range offsetsChan {
				require.Equal(t, istructs.Offset(13), offset)
			}
		}(offsetsChan)
	}
	for _, unsubscribe := range unsubscribes {
		unsubscribe()
	}
	wg.Wait()
}
