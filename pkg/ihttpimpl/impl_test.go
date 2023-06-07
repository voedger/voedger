/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin (refactoring)
 */

package ihttpimpl

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/ibus"
	"github.com/voedger/voedger/pkg/ibusmem"
	"github.com/voedger/voedger/pkg/ihttp"

	"github.com/stretchr/testify/require"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestBasicUsage_HTTPProcessor(t *testing.T) {
	require := require.New(t)
	testApp := setUp(t)
	defer tearDown(testApp)

	t.Run("deploy folder", func(t *testing.T) {
		resources := []string{
			"dir1",
			"dir2/content",
		}
		for _, res := range resources {
			dir, fileName := makeTmpContent(require, res)
			defer os.RemoveAll(dir)
			fs := os.DirFS(dir)
			err := testApp.api.DeployStaticContent(testApp.ctx, res, fs)
			require.NoError(err)

			body := testApp.get("/static/" + res + "/" + filepath.Base(fileName))
			require.Equal([]byte(filepath.Base(res)), body)
		}
	})

	t.Run("deploy embedded", func(t *testing.T) {
		testContentFS, err := fs.Sub(testContentFS, "testcontent")
		require.NoError(err)
		err = testApp.api.DeployStaticContent(testApp.ctx, "embedded", testContentFS)
		require.NoError(err)
		body := testApp.get("/static/embedded/test.txt")
		require.Equal([]byte("test file content"), body)
	})

	t.Run("404 not found on unknown resource", func(t *testing.T) {
		paths := []string{
			"/static/dir2/unknown-file",
			"/static/dir2/",
			"/static/dir2",
			"/static/dir2unknown/unknown-file",
			"/static/unknowndir/unknown-file",
			"/static/unknown-file",
			"/static/embedded/unknown",
			"/static/unknown",
			"/static",
			"/unknown",
			"/",
			"",
		}

		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				body := testApp.get(path, http.StatusNotFound)
				log.Println(string(body))
			})
		}
	})
}

//go:embed testcontent/*
var testContentFS embed.FS

type testApp struct {
	ctx           context.Context
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
	bus           ibus.IBus
	processor     ihttp.IHTTPProcessor
	api           ihttp.IHTTPProcessorAPI
	cleanups      []func()
	listeningPort int
	t             *testing.T
}

func setUp(t *testing.T) *testApp {
	require := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	// create Bus

	timeout := time.Second
	if coreutils.IsDebug() {
		timeout = time.Hour
	}
	bus, cleanup := ibusmem.New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: timeout})
	cleanups := []func(){cleanup}

	// create and start HTTPProcessor

	params := ihttp.CLIParams{
		Port: 0, // listen using some free port, port value will be taken using API
	}
	processor, pCleanup, err := NewProcessor(params, bus)
	require.NoError(err)
	cleanups = append(cleanups, pCleanup)

	err = processor.Prepare()
	require.NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		processor.Run(ctx)
	}()

	// create API

	api, err := NewAPI(bus, processor)
	require.NoError(err)

	listeningPort, err := api.ListeningPort(ctx)
	require.NoError(err)
	require.Equal(processor.(*httpProcessor).listener.Addr().(*net.TCPAddr).Port, listeningPort)

	// reverse cleanups
	for i, j := 0, len(cleanups)-1; i < j; i, j = i+1, j-1 {
		cleanups[i], cleanups[j] = cleanups[j], cleanups[i]
	}

	return &testApp{
		ctx:           ctx,
		cancel:        cancel,
		wg:            &wg,
		bus:           bus,
		processor:     processor,
		api:           api,
		cleanups:      cleanups,
		listeningPort: listeningPort,
		t:             t,
	}
}

func tearDown(ta *testApp) {
	ta.cancel()
	ta.wg.Wait()
	for _, cleanup := range ta.cleanups {
		cleanup()
	}
}

func (ta *testApp) get(resource string, expectedCodes ...int) []byte {
	require := require.New(ta.t)
	ta.t.Helper()

	url := fmt.Sprintf("http://localhost:%d%s", ta.listeningPort, resource)

	res, err := http.Get(url)
	require.NoError(err)
	expectedCode := http.StatusOK
	if len(expectedCodes) > 0 {
		expectedCode = expectedCodes[0]
	}
	require.Equal(expectedCode, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(err)

	return body
}

func makeTmpContent(require *require.Assertions, pattern string) (dir string, fileName string) {
	dir, err := os.MkdirTemp("", "."+filepath.Base(pattern))
	require.NoError(err)

	fileName = "tmpcontext.txt"

	err = os.WriteFile(filepath.Join(dir, fileName), []byte(filepath.Base(pattern)), 0644)
	require.NoError(err)

	return dir, fileName
}
