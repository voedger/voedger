/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
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
	unsubscribes := make(chan func(), cnt)
	offsetsChans := make(chan federation.OffsetsChan, cnt)
	for wsid := istructs.WSID(0); wsid < istructs.WSID(cnt); wsid++ {
		wg.Add(1)
		go func(wsid istructs.WSID) {
			defer wg.Done()
			offsetsChan, unsubscribe, err := vit.N10NSubscribe(in10n.ProjectionKey{
				App:        appdef.NewAppQName("untill", "Application"),
				Projection: appdef.NewQName("paa", "price"),
				WS:         wsid,
			})
			require.NoError(t, err)
			unsubscribes <- unsubscribe
			offsetsChans <- offsetsChan
		}(wsid)
	}
	wg.Wait()
	close(unsubscribes)
	close(offsetsChans)
	for unsubscribe := range unsubscribes {
		unsubscribe()
	}
	for offsetsChan := range offsetsChans {
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

	wg := sync.WaitGroup{}
	cnt := 10
	unsubscribes := make(chan func(), cnt)
	offsetsChans := make(chan federation.OffsetsChan, cnt)
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
			offsetsChan, unsubscribe, err := vit.N10NSubscribe(projectionKey)
			require.NoError(t, err)
			unsubscribes <- unsubscribe
			offsetsChans <- offsetsChan

			// upate
			require.NoError(t, vit.N10NUpdate(projectionKey, 13))
		}(wsid)
	}
	wg.Wait()
	close(unsubscribes)
	close(offsetsChans)
	wg = sync.WaitGroup{}
	for offsetsChan := range offsetsChans {
		wg.Add(1)
		go func(offsetsChan federation.OffsetsChan) {
			defer wg.Done()
			for offset := range offsetsChan {
				require.Equal(t, istructs.Offset(13), offset)
			}
		}(offsetsChan)
	}
	for unsubscribe := range unsubscribes {
		unsubscribe()
	}
	wg.Wait()
}
