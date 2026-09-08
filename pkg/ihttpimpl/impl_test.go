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
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	voedger "github.com/voedger/voedger/cmd/voedger/voedgerimpl"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/filesu"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/ihttp"
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
			dir, fileName := makeTmpContent(t, res)
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
		require.Equal(fmt.Sprintf(`Hello, {"text":%q}, {"par1":"val1","par2":"val2"}`, testText), string(body))

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

		body := testApp.post("/api/"+path, httpu.ContentType_ApplicationJSON, testText, nil)
		require.JSONEq(fmt.Sprintf(`{"sections":[{"type":"","elements":["Hello, %s, {}"]}]}`, testText), string(body))
	})

	t.Run("call unknown app", func(t *testing.T) {
		appOwner := "test"
		appName := uuid.New().String()
		resource := "c.SomeCommand"

		wsid := 10
		testText := "Test"
		path := fmt.Sprintf("%s/%s/%d/%s", appOwner, appName, wsid, resource)

		body := testApp.post("/api/"+path, "text/plain", testText, nil)
		require.JSONEq("{\"sys.Error\":{\"HTTPStatus\":400,\"Message\":\"app is not deployed\"}}", string(body))
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
	// Exercise route matching and proxying through the HTTP processor's real listener.
	testApp := setUp(t)
	defer tearDown(testApp)

	// Return the observation with its response so failed subtests leave no shared state.
	type upstreamRequest struct {
		URL     string
		Headers http.Header
	}
	targetServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		recorded, err := json.Marshal(upstreamRequest{URL: getFullRequestedURL(req), Headers: req.Header})
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		rw.Header().Set("X-Test-Upstream-Request", string(recorded))
		_, _ = io.WriteString(rw, "proxied")
	}))
	defer targetServer.Close()
	// The test server also owns and cleans up this client's connection pool.
	client := targetServer.Client()

	testAppURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(testApp.processor.ListeningPort()))
	testContentSubFs, err := fs.Sub(testContentFS, "testcontent")
	require.NoError(t, err)
	staticBody, err := fs.ReadFile(testContentSubFs, "test.txt")
	require.NoError(t, err)
	// Strip service prefixes; similar-looking and nested prefixes use the fallback.
	for _, prefix := range []string{"grafana", "prometheus"} {
		testApp.processor.AddReverseProxyRoute("(https?://[^/]*)/"+prefix+"($|/.*)", targetServer.URL+"$2")
	}
	testApp.processor.SetReverseProxyRouteDefault("^(https?)://([^/]+)/([^?]+)?(\\?(.+))?$", targetServer.URL+"/unknown/$3")
	testApp.processor.DeployStaticContent("embedded", testContentSubFs)

	// Static files take precedence over the fallback, including the 404 for a missing file.
	cases := []struct {
		path         string
		upstreamPath string
		status       int
		body         string
	}{
		{"/static/embedded/test.txt", "", http.StatusOK, string(staticBody)},
		{"/grafana", "/", http.StatusOK, "proxied"},
		{"/grafana/", "/", http.StatusOK, "proxied"},
		{"/grafana/report", "/report", http.StatusOK, "proxied"},
		{"/prometheus", "/", http.StatusOK, "proxied"},
		{"/prometheus/", "/", http.StatusOK, "proxied"},
		{"/prometheus/report", "/report", http.StatusOK, "proxied"},
		{"/grafanawhatever", "/unknown/grafanawhatever", http.StatusOK, "proxied"},
		{"/a/grafana", "/unknown/a/grafana", http.StatusOK, "proxied"},
		{"/a/b/grafana/whatever", "/unknown/a/b/grafana/whatever", http.StatusOK, "proxied"},
		{"/z/prometheus", "/unknown/z/prometheus", http.StatusOK, "proxied"},
		{"/z/v/prometheus/whatever", "/unknown/z/v/prometheus/whatever", http.StatusOK, "proxied"},
		{"/some_unregistered_path", "/unknown/some_unregistered_path", http.StatusOK, "proxied"},
		{"/static/embedded/test2.txt", "", http.StatusNotFound, "404 page not found\n"},
	}
	// Match Director behavior: append the peer IP and preserve other forwarding metadata.
	// SetXForwarded alone would lose the incoming chain and Forwarded. It would
	// also replace original.example/https with the upstream host/http, and
	// generate host/protocol headers in the case with no incoming headers.
	headerCases := []struct {
		name         string
		headers      http.Header
		forwardedFor string
	}{
		{name: "no incoming forwarding headers", forwardedFor: "127.0.0.1"},
		{
			name: "preserve forwarding headers and append client IP",
			headers: http.Header{
				"X-Forwarded-For":   {"192.0.2.1, 198.51.100.2"},
				"X-Forwarded-Host":  {"original.example"},
				"X-Forwarded-Proto": {"https"},
				"Forwarded":         {"for=192.0.2.1;host=original.example;proto=https"},
			},
			forwardedFor: "192.0.2.1, 198.51.100.2, 127.0.0.1",
		},
	}
	// Check both header cases for every path so header handling cannot mask routing regressions.
	for _, tc := range cases {
		for _, hc := range headerCases {
			t.Run(tc.path+"/"+hc.name, func(t *testing.T) {
				require := require.New(t)
				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testAppURL+tc.path, http.NoBody)
				require.NoError(err)
				req.Header = hc.headers.Clone()
				resp, err := client.Do(req)
				require.NoError(err)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				require.NoError(err)
				require.Equal(tc.status, resp.StatusCode)
				require.Equal(tc.body, string(body))

				recorded := resp.Header.Get("X-Test-Upstream-Request")
				if tc.upstreamPath == "" {
					require.Empty(recorded, "static requests must not reach the upstream")
					require.Equal(testAppURL+tc.path, resp.Request.URL.String())
					return
				}
				// This snapshot belongs to this response; there is nothing to wait for or drain.
				require.NotEmpty(recorded, "request must reach the upstream")
				var actual upstreamRequest
				require.NoError(json.Unmarshal([]byte(recorded), &actual))
				require.Equal(targetServer.URL+tc.upstreamPath, actual.URL)
				require.Equal([]string{hc.forwardedFor}, actual.Headers.Values("X-Forwarded-For"))
				// Unset forwarding metadata must stay unset; supplied values must survive.
				for _, name := range []string{"X-Forwarded-Host", "X-Forwarded-Proto", "Forwarded"} {
					require.Equal(hc.headers.Values(name), actual.Headers.Values(name), name)
				}
			})
		}
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
	wg.Go(func() {
		for i := range 1000 {
			testApp.processor.DeployStaticContent(fmt.Sprintf("test_path_%d", i), testContentSubFs)
		}
	})

	wg.Go(func() {
		for i := range 1000 {
			testApp.get(fmt.Sprintf("/test_path_%d", i), []int{http.StatusOK, http.StatusNotFound}...)
		}
	})

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
	appStorageProvider := istorageimpl.Provide(mem.Provide(testingu.MockTime))
	routerStorage, err := ihttp.NewIRouterStorage(appStorageProvider)
	require.NoError(err)
	processor, pCleanup := NewProcessor(params, routerStorage)
	cleanups = append(cleanups, pCleanup)

	err = processor.Prepare()
	require.NoError(err)

	wg := sync.WaitGroup{}
	wg.Go(func() {
		processor.Run(ctx)
	})

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
		var err error
		requestData, err = json.Marshal(requestMap)
		if err != nil {
			// notest
			panic(err)
		}
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestData))
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

func makeTmpContent(t *testing.T, pattern string) (dir string, fileName string) {
	t.Helper()
	dir = t.TempDir()
	fileName = "tmpcontext.txt"
	require.NoError(t, os.WriteFile(filepath.Join(dir, fileName), []byte(filepath.Base(pattern)), filesu.FileMode_DefaultForFile))
	return dir, fileName
}
