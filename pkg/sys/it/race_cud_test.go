/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

// The goal of package is  to ensure there are no Race Condition/Race Data errors in Voedger read/write operations
// All tests should be run with -race
package sys_it

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
)

const (
	readCnt  = 10
	writeCnt = 20
)

var cfg it.VITConfig

// nolint
func init() {
	appOpts := []it.AppOptFunc{
		it.WithWorkspaceTemplate(it.QNameApp1_TestWSKind, "test_template", sys_test_template.TestTemplateFS),
		it.WithUserLogin("login", "pwd"),
	}
	for i := 0; i < writeCnt; i++ {
		appOpts = append(appOpts, it.WithChildWorkspace(it.QNameApp1_TestWSKind, "test_ws_"+strconv.Itoa(i), "test_template", "", "login", map[string]interface{}{"IntFld": 42}))
	}
	cfg = it.NewSharedVITConfig(it.WithApp(istructs.AppQName_test1_app1, it.ProvideApp1, appOpts...))
}

// One WSID
//*****************************************

// Read from many goroutines.
// Read result does not matter.

func Test_Race_CUDSimpleRead(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	cnt := readCnt
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			writeArt(ws, vit)
			readArt(vit, ws)
		}()
	}
	wg.Wait()
}

// Write from many goroutines
func Test_Race_CUDSimpleWrite(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	cnt := writeCnt
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			writeArt(ws, vit)
		}()
	}
	wg.Wait()
}
func Test_Race_CUDOneWriteManyRead(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	wg := sync.WaitGroup{}
	wg.Add(1)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	go func() {
		defer wg.Done()
		writeArt(ws, vit)
	}()

	for i := 0; i < readCnt; i++ {
		wg.Add(1)
		go func(_ *testing.T, _ int) {
			defer wg.Done()
			readArt(vit, ws)
		}(t, i)
	}
	wg.Wait()
}

// Write from many goroutines, one read after all writes are finished
// Read result: only status = OK checked
func Test_Race_CUDManyWriteOneRead(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	cnt := writeCnt
	wg := sync.WaitGroup{}
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			writeArt(ws, vit)
		}()
	}
	wg.Wait()

	wgr := sync.WaitGroup{}
	wgr.Add(1)
	go func(_ *testing.T) {
		defer wgr.Done()
		readArt(vit, ws)
	}(t)
	wgr.Wait()
}

// Write from many goroutines, and simultaneous read from many goroutines
// Read result: only status = OK checked
func Test_Race_CUDManyWriteManyReadNoResult(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	cnt := writeCnt
	wg := sync.WaitGroup{}
	wg.Add(2 * cnt)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			writeArt(ws, vit)
		}()
		go func() {
			defer wg.Done()
			readArt(vit, ws)
		}()
	}
	wg.Wait()
}

// Write from many goroutines & read from many goroutines after all data has been written
// Read result: Checks all written data are correct
func Test_Race_CUDManyWriteManyReadCheckResult(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	cnt := writeCnt
	wgW := sync.WaitGroup{}
	wgW.Add(cnt)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	artNumbers := make(chan int, cnt)
	for i := 0; i < cnt; i++ {
		go func() {
			defer wgW.Done()
			artNumbers <- writeArt(ws, vit)
		}()
	}
	wgW.Wait()
	close(artNumbers)

	wgR := sync.WaitGroup{}
	wgR.Add(cnt)
	for i := 0; i < cnt; i++ {
		go func(at *testing.T) {
			defer wgR.Done()
			readAndCheckArt(at, <-artNumbers, vit, ws)
		}(t)
	}
	wgR.Wait()
}

// Write from many goroutines
// Read & Update from many goroutines with different pauses after all data has been written
// Read result: only status = OK checked
func Test_Race_CUDManyUpdateManyReadCheckResult(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	cntw := writeCnt
	wgW := sync.WaitGroup{}
	wgW.Add(cntw)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	artNumbers := make(chan int, cntw)
	for i := 0; i < cntw; i++ {
		go func() {
			defer wgW.Done()
			artNumbers <- writeArt(ws, vit)
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
				updateArtByName(<-artNumbers, acnt, vit, ws)
			}(cntw)
		}

		cntr := writeCnt
		for i := 0; i < cntr; i++ {
			wgUR.Add(1)
			go func() {
				defer wgUR.Done()
				readArt(vit, ws)
			}()
		}

		wgUR.Wait()
	}
}

// Many WSIDs
//*****************************************

// Read from many goroutines with different WSID
func Test_Race_CUDManyReadCheckResult(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	cntWS := readCnt

	wg := sync.WaitGroup{}
	for prtIdx := istructs.WSID(1); int(prtIdx) < cntWS; prtIdx++ {
		wg.Add(1)
		go func(wsidNum istructs.WSID) {
			defer wg.Done()
			ws := vit.WS(istructs.AppQName_test1_app1, "test_ws_"+strconv.Itoa(int(wsidNum)))
			readArt(vit, ws)
		}(prtIdx)
	}
	wg.Wait()
}

// Write from many goroutines with different WSID
func Test_Race_CUDManyWriteCheckResult(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	cntWS := writeCnt
	var prtIdx istructs.WSID

	wg := sync.WaitGroup{}
	for prtIdx = 1; int(prtIdx) < cntWS; prtIdx++ {
		wg.Add(1)
		go func(wsidNum istructs.WSID) {
			defer wg.Done()
			ws := vit.WS(istructs.AppQName_test1_app1, "test_ws_"+strconv.Itoa(int(wsidNum)))
			writeArt(ws, vit)
		}(prtIdx)
	}
	wg.Wait()
}

// Read & Write from many goroutines with different WSID
func Test_Race_CUDManyWriteReadCheckResult(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if coreutils.IsCassandraStorage() {
		return
	}
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	cntWS := writeCnt
	var prtIdx istructs.WSID

	for k := 1; k < 10; k++ {
		wg := sync.WaitGroup{}
		for prtIdx = 1; int(prtIdx) < cntWS; prtIdx++ {
			wg.Add(1)
			go func(wsidNum istructs.WSID) {
				defer wg.Done()
				ws := vit.WS(istructs.AppQName_test1_app1, "test_ws_"+strconv.Itoa(int(wsidNum)))
				writeArt(ws, vit)
			}(prtIdx)
		}
		for prtIdx = 1; int(prtIdx) < cntWS; prtIdx++ {
			wg.Add(1)
			go func(wsidNum istructs.WSID) {
				defer wg.Done()
				ws := vit.WS(istructs.AppQName_test1_app1, "test_ws_"+strconv.Itoa(int(wsidNum)))
				readArt(vit, ws)
			}(prtIdx)
		}
		wg.Wait()
	}
}

func writeArt(ws *it.AppWorkspace, vit *it.VIT) (artNumber int) {
	artNumber = vit.NextNumber()
	idstr := strconv.Itoa(artNumber)
	artname := "cola" + idstr
	body := `
		{
			"cuds": [
				{
					"fields": {
						"sys.ID": ` + idstr + `,
						"sys.QName": "app1pkg.articles",
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
	vit.PostWS(ws, "c.sys.CUD", body)
	return
}

func readArt(vit *it.VIT, ws *it.AppWorkspace) *coreutils.FuncResponse {
	body := `
	{
		"args":{
			"Schema":"app1pkg.articles"
		},
		"elements":[
			{
				"fields": ["name", "control_active", "sys.ID"]
			}
		],
		"orderBy":[{"field":"name"}]
	}`
	return vit.PostWS(ws, "q.sys.Collection", body)
}

func updateArtByName(idx, num int, vit *it.VIT, ws *it.AppWorkspace) {
	artname := "cola" + strconv.Itoa(idx)
	resp := readArt(vit, ws)

	var actualName string
	for i := 0; i < num; i++ {
		actualName = resp.SectionRow(i)[0].(string)
		if artname == actualName {
			id := resp.SectionRow()[2].(float64)
			updateArt(id, vit, ws)
			break
		}
	}
}

func updateArt(id float64, vit *it.VIT, ws *it.AppWorkspace) {
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
	vit.PostWS(ws, "c.sys.CUD", body)
}

func readAndCheckArt(t *testing.T, idx int, vit *it.VIT, ws *it.AppWorkspace) {
	idstr := strconv.Itoa(idx)
	artname := "cola" + idstr
	require := require.New(t)
	var id float64

	resp := readArt(vit, ws)

	var actualName string
	var actualControlActive float64
	i := 0
	for artname != actualName {
		actualName = resp.SectionRow(i)[0].(string)
		actualControlActive = resp.SectionRow(i)[1].(float64)
		id = resp.SectionRow(i)[2].(float64)
		require.NotEqual(0, id)
		i++
	}
	require.Equal(artname, actualName)
	require.Equal(float64(5), actualControlActive)
}
