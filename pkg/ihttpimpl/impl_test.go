/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin (refactoring)
 * @author Alisher Nurmanov
 */

package ihttpimpl

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
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

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	voedger "github.com/voedger/voedger/cmd/voedger/voedgerimpl"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/ihttpctl"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
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
			dirFS := os.DirFS(dir)
			testApp.processor.DeployStaticContent(res, dirFS)

			body := testApp.get("/static/" + res + "/" + filepath.Base(fileName))
			require.Equal([]byte(filepath.Base(res)), body)
		}
	})

	t.Run("deploy embedded", func(t *testing.T) {
		testContentFS, err := fs.Sub(testContentFS, "testcontent")
		require.NoError(err)
		testApp.processor.DeployStaticContent("embedded", testContentFS)
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

	t.Run("c.EchoCommand", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 10, 1)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployApp(testAppQName)
		}()

		err = testApp.processor.DeployAppPartition(testAppQName, 4, voedger.NewSysRouterRequestHandler)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployAppPartition(testAppQName, 0)
		}()

		wsid := 10
		testText := "Test"
		resource := "c.EchoCommand"
		path := fmt.Sprintf("%s/%s/%d/%s?par1=val1&par2=val2", appOwner, appName, wsid, resource)

		body := testApp.post("/api/"+path, "text/plain", testText, nil)
		require.Equal(`Hello, Test, {"par1":"val1","par2":"val2"}`, string(body))

		body = testApp.post("/api/"+path, "application/json", "", map[string]string{"text": testText})
		require.Equal(fmt.Sprintf(`Hello, {"text":"%s"}, {"par1":"val1","par2":"val2"}`, testText), string(body))

		testText = ""
		body = testApp.post("/api/"+path, "text/plain", testText, nil)
		require.Equal(fmt.Sprintf(`Hello, %s, {"par1":"val1","par2":"val2"}`, testText), string(body))
	})

	t.Run("q.EchoQuery", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 10, 1)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployApp(testAppQName)
		}()

		err = testApp.processor.DeployAppPartition(testAppQName, 4, voedger.NewSysRouterRequestHandler)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployAppPartition(testAppQName, 0)
		}()

		wsid := 10
		testText := "Test"
		resource := "q.EchoQuery"
		path := fmt.Sprintf("%s/%s/%d/%s", appOwner, appName, wsid, resource)

		body := testApp.post("/api/"+path, coreutils.ApplicationJSON, testText, nil)
		require.Equal(fmt.Sprintf(`{"sections":[{"type":"","elements":["Hello, %s, {}"]}]}`, testText), string(body))
	})

	t.Run("call unknown app", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		resource := "c.SomeCommand"

		wsid := 10
		testText := "Test"
		path := fmt.Sprintf("%s/%s/%d/%s", appOwner, appName, wsid, resource)

		body := testApp.post("/api/"+path, "text/plain", testText, nil)
		require.Equal([]byte("{\"sys.Error\":{\"HTTPStatus\":400,\"Message\":\"app is not deployed\"}}"), body)
	})

	t.Run("deploy the same app twice", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 10, 1)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployApp(testAppQName)
		}()

		err = testApp.processor.DeployApp(testAppQName, 10, 1)
		require.ErrorIs(err, ErrAppAlreadyDeployed)

	})

	t.Run("undeploy not deployed yet app", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 10, 1)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployApp(testAppQName)
		}()

		unknownAppName := uuid.New().String()
		err = testApp.processor.UndeployApp(appdef.NewAppQName(appOwner, unknownAppName))
		require.ErrorIs(err, ErrAppIsNotDeployed)

	})

	t.Run("undeploy app part which is not deployed yet", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 10, 1)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployApp(testAppQName)
		}()

		err = testApp.processor.UndeployAppPartition(testAppQName, 0)
		require.ErrorIs(err, ErrAppPartitionIsNotDeployed)

	})

	t.Run("undeploy wrong app part no", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 10, 1)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployApp(testAppQName)
		}()

		err = testApp.processor.DeployAppPartition(testAppQName, 0, voedger.NewSysRouterRequestHandler)
		require.NoError(err)

		err = testApp.processor.UndeployAppPartition(testAppQName, 1)
		require.ErrorIs(err, ErrAppPartitionIsNotDeployed)

	})

	t.Run("app part no is out of range", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 2, 1)
		require.NoError(err)

		defer func() {
			_ = testApp.processor.UndeployApp(testAppQName)
		}()

		err = testApp.processor.DeployAppPartition(testAppQName, 3, voedger.NewSysRouterRequestHandler)
		require.ErrorIs(err, ErrAppPartNoOutOfRange)
	})

	t.Run("undeploy active app part", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		testAppQName := appdef.NewAppQName(appOwner, appName)
		err := testApp.processor.DeployApp(testAppQName, 2, 1)
		require.NoError(err)

		err = testApp.processor.DeployAppPartition(testAppQName, 0, voedger.NewSysRouterRequestHandler)
		require.NoError(err)

		err = testApp.processor.UndeployApp(testAppQName)
		require.ErrorIs(err, ErrActiveAppPartitionsExist)
	})
}

func TestReverseProxy(t *testing.T) {
	require := require.New(t)
	testApp := setUp(t)
	defer tearDown(testApp)

	testAppPort := testApp.processor.ListeningPort()
	targetListener, err := net.Listen("tcp", coreutils.ServerAddress(0))
	require.NoError(err)
	targetListenerPort := targetListener.Addr().(*net.TCPAddr).Port

	errs := make(chan error)
	defer close(errs)

	paths := map[string]string{
		"/static/embedded/test.txt":  fmt.Sprintf("http://127.0.0.1:%d/static/embedded/test.txt", testAppPort),
		"/grafana":                   fmt.Sprintf("http://127.0.0.1:%d/", targetListenerPort),
		"/grafana/":                  fmt.Sprintf("http://127.0.0.1:%d/", targetListenerPort),
		"/grafana/report":            fmt.Sprintf("http://127.0.0.1:%d/report", targetListenerPort),
		"/prometheus":                fmt.Sprintf("http://127.0.0.1:%d/", targetListenerPort),
		"/prometheus/":               fmt.Sprintf("http://127.0.0.1:%d/", targetListenerPort),
		"/prometheus/report":         fmt.Sprintf("http://127.0.0.1:%d/report", targetListenerPort),
		"/grafanawhatever":           fmt.Sprintf("http://127.0.0.1:%d/unknown/grafanawhatever", targetListenerPort),
		"/a/grafana":                 fmt.Sprintf("http://127.0.0.1:%d/unknown/a/grafana", targetListenerPort),
		"/a/b/grafana/whatever":      fmt.Sprintf("http://127.0.0.1:%d/unknown/a/b/grafana/whatever", targetListenerPort),
		"/z/prometheus":              fmt.Sprintf("http://127.0.0.1:%d/unknown/z/prometheus", targetListenerPort),
		"/z/v/prometheus/whatever":   fmt.Sprintf("http://127.0.0.1:%d/unknown/z/v/prometheus/whatever", targetListenerPort),
		"/some_unregistered_path":    fmt.Sprintf("http://127.0.0.1:%d/unknown/some_unregistered_path", targetListenerPort),
		"/static/embedded/test2.txt": fmt.Sprintf("http://127.0.0.1:%d/static/embedded/test2.txt", testAppPort),
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

	testRedirectionRoutes := func() ihttpctl.RedirectRoutes {
		return ihttpctl.RedirectRoutes{
			"(https?://[^/]*)/grafana($|/.*)":    fmt.Sprintf("http://127.0.0.1:%d$2", targetListenerPort),
			"(https?://[^/]*)/prometheus($|/.*)": fmt.Sprintf("http://127.0.0.1:%d$2", targetListenerPort),
		}
	}

	for srcRegExp, dstRegExp := range testRedirectionRoutes() {
		testApp.processor.AddReverseProxyRoute(srcRegExp, dstRegExp)
	}
	testApp.processor.SetReverseProxyRouteDefault("^(https?)://([^/]+)/([^?]+)?(\\?(.+))?$", fmt.Sprintf("http://127.0.0.1:%d/unknown/$3", targetListenerPort))
	testApp.processor.DeployStaticContent("embedded", testContentSubFs)
	for requestedPath, expectedPath := range paths {
		targetHandler.expectedURLPath = expectedPath
		testApp.get(requestedPath)
	}
}

func TestRace_HTTPProcessor(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)
	testApp := setUp(t)
	defer tearDown(testApp)

	testContentSubFs, err := fs.Sub(testContentFS, "testcontent")
	require.NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 1000; i++ {
			testApp.processor.DeployStaticContent(fmt.Sprintf("test_path_%d", i), testContentSubFs)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 1000; i++ {
			testApp.get(fmt.Sprintf("/test_path_%d", i), []int{http.StatusOK, http.StatusNotFound}...)
		}
	}()

	wg.Wait()
}

//go:embed testcontent/*
var testContentFS embed.FS

type testApp struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        *sync.WaitGroup
	processor ihttp.IHTTPProcessor
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
	appStorageProvider := istorageimpl.Provide(mem.Provide(coreutils.MockTime))
	routerStorage, err := ihttp.NewIRouterStorage(appStorageProvider)
	require.NoError(err)
	processor, pCleanup := NewProcessor(params, routerStorage)
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

	// reverse cleanups
	for i, j := 0, len(cleanups)-1; i < j; i, j = i+1, j-1 {
		cleanups[i], cleanups[j] = cleanups[j], cleanups[i]
	}

	return &testApp{
		ctx:       ctx,
		cancel:    cancel,
		wg:        &wg,
		processor: processor,
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
	if len(expectedCodes) > 0 {
		require.Contains(expectedCodes, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	require.NoError(err)
	err = res.Body.Close()
	require.NoError(err)

	return body
}

func (ta *testApp) post(resource string, contentType string, requestText string, requestMap map[string]string) []byte {
	require := require.New(ta.t)
	ta.t.Helper()

	url := fmt.Sprintf("http://localhost:%d%s", ta.processor.ListeningPort(), resource)

	var requestData []byte
	if requestText != "" {
		requestData = []byte(requestText)
	}
	if requestMap != nil {
		requestData, _ = json.Marshal(requestMap)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestData))
	require.NoError(err)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(err)
	err = resp.Body.Close()
	require.NoError(err)

	return body
}

func makeTmpContent(require *require.Assertions, pattern string) (dir string, fileName string) {
	dir, err := os.MkdirTemp("", "."+filepath.Base(pattern))
	require.NoError(err)

	fileName = "tmpcontext.txt"

	err = os.WriteFile(filepath.Join(dir, fileName), []byte(filepath.Base(pattern)), coreutils.FileMode_rw_rw_rw_)
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
