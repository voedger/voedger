/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

// The goal of package is  to ensure there are no Race Condition/Race Data errors in Heeus read/write operations
// All tests should be run with -race
package heeus_it

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	airsbp_it "github.com/untillpro/airs-bp3/packages/air/it"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

const (
	readCnt  = 10
	writeCnt = 20
)

// One WSID
//*****************************************

// Read from many goroutines.
// Read result does not matter.
func Test_Race_CUDSimpleRead(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	cnt := readCnt
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			readArt(hit, ws)
		}()
	}
	wg.Wait()
}

// Write from many goroutines
func Test_Race_CUDSimpleWrite(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	cnt := writeCnt
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			writeArt(ws, hit)
		}()
	}
	wg.Wait()
}
func Test_Race_CUDOneWriteManyRead(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	wg := sync.WaitGroup{}
	wg.Add(1)
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	go func() {
		defer wg.Done()
		writeArt(ws, hit)
	}()

	for i := 0; i < readCnt; i++ {
		wg.Add(1)
		go func(_ *testing.T, _ int) {
			defer wg.Done()
			readArt(hit, ws)
		}(t, i)
	}
	wg.Wait()
}

// Write from many goroutines, one read after all writes are finished
// Read result: only status = OK checked
func Test_Race_CUDManyWriteOneRead(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	cnt := writeCnt
	wg := sync.WaitGroup{}
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			writeArt(ws, hit)
		}()
	}
	wg.Wait()

	wgr := sync.WaitGroup{}
	wgr.Add(1)
	go func(_ *testing.T) {
		defer wgr.Done()
		readArt(hit, ws)
	}(t)
	wgr.Wait()
}

// Write from many goroutines, and simultaneous read from many goroutines
// Read result: only status = OK checked
func Test_Race_CUDManyWriteManyReadNoResult(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	cnt := writeCnt
	wg := sync.WaitGroup{}
	wg.Add(2 * cnt)
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			writeArt(ws, hit)
		}()
		go func() {
			defer wg.Done()
			readArt(hit, ws)
		}()
	}
	wg.Wait()
}

// Write from many goroutines & read from many goroutines after all data has been written
// Read result: Checks all written data are correct
func Test_Race_CUDManyWriteManyReadCheckResult(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	cnt := writeCnt
	wgW := sync.WaitGroup{}
	wgW.Add(cnt)
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	artNumbers := make(chan int, cnt)
	for i := 0; i < cnt; i++ {
		go func() {
			defer wgW.Done()
			artNumbers <- writeArt(ws, hit)
		}()
	}
	wgW.Wait()
	close(artNumbers)

	wgR := sync.WaitGroup{}
	wgR.Add(cnt)
	for i := 0; i < cnt; i++ {
		go func(at *testing.T) {
			defer wgR.Done()
			readAndCheckArt(at, <-artNumbers, hit, ws)
		}(t)
	}
	wgR.Wait()
}

// Write from many goroutines
// Read & Update from many goroutines with different pauses after all data has been written
// Read result: only status = OK checked
func Test_Race_CUDManyUpdateManyReadCheckResult(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	cntw := writeCnt
	wgW := sync.WaitGroup{}
	wgW.Add(cntw)
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	artNumbers := make(chan int, cntw)
	for i := 0; i < cntw; i++ {
		go func() {
			defer wgW.Done()
			artNumbers <- writeArt(ws, hit)
		}()
	}
	wgW.Wait()
	close(artNumbers)

	for k := 0; k < 5; k++ {
		wgUR := sync.WaitGroup{}
		for i := 0; i < cntw; i++ {
			wgUR.Add(1)
			go func(acnt int) {
				defer wgUR.Done()
				updateArtByName(<-artNumbers, acnt, hit, ws)
			}(cntw)
		}

		cntr := writeCnt
		for i := 0; i < cntr; i++ {
			wgUR.Add(1)
			go func() {
				defer wgUR.Done()
				readArt(hit, ws)
			}()
		}

		wgUR.Wait()
	}
}

// Many WSIDs
//*****************************************

// Read from many goroutines with different WSID
func Test_Race_CUDManyReadCheckResult(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	var cntWS int = readCnt

	wg := sync.WaitGroup{}
	for prtIdx := istructs.WSID(1); int(prtIdx) < cntWS; prtIdx++ {
		wg.Add(1)
		go func(wsid istructs.WSID) {
			defer wg.Done()
			ws := hit.DummyWS(istructs.AppQName_untill_airs_bp, wsid)
			readArt(hit, ws)
		}(prtIdx)
	}
	wg.Wait()
}

// Write from many goroutines with different WSID
func Test_Race_CUDManyWriteCheckResult(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	var cntWS int = writeCnt
	var prtIdx istructs.WSID

	wg := sync.WaitGroup{}
	for prtIdx = 1; int(prtIdx) < cntWS; prtIdx++ {
		wg.Add(1)
		go func(wsid istructs.WSID) {
			defer wg.Done()
			ws := hit.DummyWS(istructs.AppQName_untill_airs_bp, wsid+istructs.MaxPseudoBaseWSID)
			writeArt(ws, hit)
		}(prtIdx)
	}
	wg.Wait()
}

// Read & Write from many goroutines with different WSID
func Test_Race_CUDManyWriteReadCheckResult(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	var cntWS int = writeCnt
	var prtIdx istructs.WSID

	for k := 1; k < 10; k++ {
		wg := sync.WaitGroup{}
		for prtIdx = 1; int(prtIdx) < cntWS; prtIdx++ {
			wg.Add(1)
			go func(wsid istructs.WSID) {
				defer wg.Done()
				ws := hit.DummyWS(istructs.AppQName_untill_airs_bp, wsid+istructs.MaxPseudoBaseWSID)
				writeArt(ws, hit)
			}(prtIdx)
		}
		for prtIdx = 1; int(prtIdx) < cntWS; prtIdx++ {
			wg.Add(1)
			go func(wsid istructs.WSID) {
				defer wg.Done()
				ws := hit.DummyWS(istructs.AppQName_untill_airs_bp, wsid+istructs.MaxPseudoBaseWSID)
				readArt(hit, ws)
			}(prtIdx)
		}
		wg.Wait()
	}
}

func writeArt(ws *it.AppWorkspace, hit *it.HIT) (artNumber int) {
	artNumber = hit.NextNumber()
	idstr := strconv.Itoa(artNumber)
	artname := "cola" + idstr
	body := `
		{
			"cuds": [
				{
					"fields": {
						"sys.ID": ` + idstr + `,
						"sys.QName": "untill.articles",
						"name": "` + artname + `",
						"article_manual": 1,
						"article_hash": 2,
						"hideonhold": 3,
						"time_active": 4,
						"control_active": 5
					}
				}
			]
		}`
	hit.PostWS(ws, "c.sys.CUD", body)
	return
}

func readArt(hit *it.HIT, ws *it.AppWorkspace) *utils.FuncResponse {
	body := `
	{
		"args":{
			"Schema":"untill.articles"
		},
		"elements":[
			{
				"fields": ["name", "control_active", "sys.ID"]
			}
		],
		"orderBy":[{"field":"name"}]
	}`
	return hit.PostWS(ws, "q.sys.Collection", body)
}

func updateArtByName(idx, num int, hit *it.HIT, ws *it.AppWorkspace) {
	artname := "cola" + strconv.Itoa(idx)
	resp := readArt(hit, ws)

	var actualName string
	for i := 0; i < num; i++ {
		actualName = resp.SectionRow(i)[0].(string)
		if artname == actualName {
			id := resp.SectionRow()[2].(float64)
			updateArt(id, hit, ws)
			break
		}
	}
}

func updateArt(id float64, hit *it.HIT, ws *it.AppWorkspace) {
	body := fmt.Sprintf(`
	{
		"cuds": [
			{
				"sys.ID": %d,
				"fields": {
					"article_manual": 110,
					"article_hash": 210,
					"hideonhold": 310,
					"time_active": 410,
					"control_active": 510
				}
			}
		]
	}`, int64(id))
	hit.PostWS(ws, "c.sys.CUD", body)
}

func readAndCheckArt(t *testing.T, idx int, hit *it.HIT, ws *it.AppWorkspace) {
	idstr := strconv.Itoa(idx)
	artname := "cola" + idstr
	require := require.New(t)
	var id float64

	resp := readArt(hit, ws)

	var actualName string
	var actualControlActive float64
	i := 0
	for artname != actualName {
		actualName = resp.SectionRow(i)[0].(string)
		actualControlActive = resp.SectionRow(i)[1].(float64)
		id = resp.SectionRow(i)[2].(float64)
		require.NotEqual(id, 0)
		i++
	}
	require.Equal(artname, actualName)
	require.Equal(float64(5), actualControlActive)
}
