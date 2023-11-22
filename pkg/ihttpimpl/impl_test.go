/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin (refactoring)
 * @author Alisher Nurmanov
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

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/ihttp"
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
			testApp.api.DeployStaticContent(res, fs)

			body := testApp.get("/static/" + res + "/" + filepath.Base(fileName))
			require.Equal([]byte(filepath.Base(res)), body)
		}
	})

	t.Run("deploy embedded", func(t *testing.T) {
		testContentFS, err := fs.Sub(testContentFS, "testcontent")
		require.NoError(err)
		testApp.api.DeployStaticContent("embedded", testContentFS)
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

func TestReverseProxy(t *testing.T) {
	require := require.New(t)
	testApp := setUp(t)
	defer tearDown(testApp)

	testAppPort := testApp.processor.ListeningPort()
	targetListenerPort := 10000
	targetListener, err := net.Listen("tcp", coreutils.ServerAddress(targetListenerPort))
	require.NoError(err)

	errs := make(chan error)
	defer close(errs)

	paths := map[string]string{
		"/static/embedded/test.txt": fmt.Sprintf("http://127.0.0.1:%d/static/embedded/test.txt", testAppPort),
		"/grafana/report":           fmt.Sprintf("http://127.0.0.1:%d/report", targetListenerPort),
		"/grafana":                  fmt.Sprintf("http://127.0.0.1:%d/", targetListenerPort),
		"/grafanawhatever":          fmt.Sprintf("http://127.0.0.1:%d/unknown/grafanawhatever", targetListenerPort),
		"/a/grafana":                fmt.Sprintf("http://127.0.0.1:%d/unknown/a/grafana", targetListenerPort),
		"/a/b/grafana/whatever":     fmt.Sprintf("http://127.0.0.1:%d/unknown/a/b/grafana/whatever", targetListenerPort),
		"/some_unregistered_path":   fmt.Sprintf("http://127.0.0.1:%d/unknown/some_unregistered_path", targetListenerPort),
	}

	targetHandler := targetHandler{t: t}
	targetServer := http.Server{
		Handler: &targetHandler,
	}
	// target server's goroutine
	go func() {
		errs <- targetServer.Serve(targetListener)
	}()

	testContentSubFs, err := fs.Sub(testContentFS, "testcontent")
	require.NoError(err)

	testApp.api.AddReverseProxyRoute("(https?://[^/]*)/grafana($|/.*)", fmt.Sprintf("http://127.0.0.1:%d$2", targetListenerPort))
	testApp.api.AddReverseProxyRouteDefault("^(https?)://([^/]+)/([^?]+)?(\\?(.+))?$", fmt.Sprintf("http://127.0.0.1:%d/unknown/$3", targetListenerPort))
	testApp.api.DeployStaticContent("embedded", testContentSubFs)
	for requestedPath, expectedPath := range paths {
		targetHandler.expectedURLPath = expectedPath
		testApp.get(requestedPath)
	}
}

//go:embed testcontent/*
var testContentFS embed.FS

type testApp struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        *sync.WaitGroup
	processor ihttp.IHTTPProcessor
	api       ihttp.IHTTPProcessorAPI
	cleanups  []func()
	t         *testing.T
}

func setUp(t *testing.T) *testApp {
	require := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	// create Bus

	cleanups := []func(){}

	// create and start HTTPProcessor

	params := ihttp.CLIParams{
		Port: 0, // listen using some free port, port value will be taken using API
	}
	processor, pCleanup, err := NewProcessor(params)
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

	api, err := NewAPI(processor)
	require.NoError(err)

	// reverse cleanups
	for i, j := 0, len(cleanups)-1; i < j; i, j = i+1, j-1 {
		cleanups[i], cleanups[j] = cleanups[j], cleanups[i]
	}

	return &testApp{
		ctx:       ctx,
		cancel:    cancel,
		wg:        &wg,
		processor: processor,
		api:       api,
		cleanups:  cleanups,
		t:         t,
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

	url := fmt.Sprintf("http://localhost:%d%s", ta.processor.ListeningPort(), resource)

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

type targetHandler struct {
	t               *testing.T
	expectedURLPath string
}

func (h *targetHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	_, err := io.ReadAll(req.Body)
	require.NoError(h.t, err)
	req.Close = true
	req.Body.Close()
	require.Equal(h.t, h.expectedURLPath, getFullRequestedURL(req))
}
