/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestBasicUsage_ReverseProxy(t *testing.T) {
	// Buffer the upstream request so the handler can reply before the subtest receives it.
	receivedRequests := make(chan *http.Request, 1)
	targetServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		select {
		case receivedRequests <- req.Clone(req.Context()):
		default:
			// A failed case must not leave later handlers blocked on a full channel.
			http.Error(rw, "previous upstream observation was not consumed", http.StatusInternalServerError)
			return
		}
		// Echoing the body checks both request delivery and response forwarding.
		_, _ = io.WriteString(rw, "hello "+string(body))
	}))
	defer targetServer.Close()

	// Point all four route types at the test upstream. The rewrite target includes a query
	// parameter so we can verify that target and incoming query strings are combined.
	cfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app1, it.ProvideApp1),
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			cfg.Routes["/grafana"] = targetServer.URL
			cfg.RoutesRewrite["/grafana-rewrite"] = targetServer.URL + "/rewritten?target=value"
			cfg.RouteDefault = targetServer.URL + "/not-found"
			cfg.RouteDomains["localhost"] = targetServer.URL
		}),
	)
	// Start a VVM with these routes to exercise its actual HTTP listener and router configuration.
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()
	// The test server also owns and cleans up this client's connection pool.
	client := targetServer.Client()

	// Match Director behavior: append the peer IP and preserve other forwarding metadata.
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
	// Verify path preservation, prefix replacement, fallback routing, and query forwarding.
	cases := []struct {
		name          string
		path          string
		host          string
		expectedPath  string
		expectedQuery string
	}{
		{"ordinary route", "grafana/foo?query=value", "", "/grafana/foo", "query=value"},
		{"trailing slash", "grafana/foo/bar/", "", "/grafana/foo/bar/", ""},
		{"rewritten route", "grafana-rewrite/foo?query=value", "", "/rewritten/foo", "target=value&query=value"},
		{"default route", "unknown/foo?query=value", "", "/not-found/unknown/foo", "query=value"},
		// https://dev.untill.com/projects/#!627070
		// Keep the empty path segment intact when an incomplete API path falls back to the upstream.
		{"double slash", "api/untill/airs-bp//c.air.StoreSubscriptionProfile", "", "/not-found/api/untill/airs-bp//c.air.StoreSubscriptionProfile", ""},
		// A domain match takes precedence over the matching path-rewrite rule, preserving the original path.
		{"domain route", "grafana-rewrite/foo/?query=value", "localhost", "/grafana-rewrite/foo/", "query=value"},
	}
	// Run both header cases for every route to verify that routing and header compatibility work together.
	for _, tc := range cases {
		for _, hc := range headerCases {
			t.Run(tc.name+"/"+hc.name, func(t *testing.T) {
				require := require.New(t)
				req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, vit.URLStr()+"/"+tc.path, strings.NewReader("world"))
				require.NoError(err)
				req.Header = hc.headers.Clone()
				if tc.host != "" {
					// Select the domain route via HTTP Host while keeping the connection directed at the local VVM.
					req.Host = tc.host
				}
				resp, err := client.Do(req)
				require.NoError(err)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				// Capture precedes the response; drain it before assertions can end this case.
				var actual *http.Request
				select {
				case actual = <-receivedRequests:
				default:
				}
				require.NoError(err)

				// Verify that the POST body reached the upstream and its reply was relayed to the client.
				require.Equal(http.StatusOK, resp.StatusCode)
				require.Equal("hello world", string(body))
				require.NotNil(actual, "request must reach the upstream")
				// Preserve the method while rewriting the host, path, and query according to the selected route.
				require.Equal(http.MethodPost, actual.Method)
				require.Equal(strings.TrimPrefix(targetServer.URL, "http://"), actual.Host)
				require.Equal(tc.expectedPath, actual.URL.Path)
				require.Equal(tc.expectedQuery, actual.URL.RawQuery)
				require.Equal([]string{hc.forwardedFor}, actual.Header.Values("X-Forwarded-For"))
				// Unset forwarding metadata must stay unset; supplied values must survive.
				for _, name := range []string{"X-Forwarded-Host", "X-Forwarded-Proto", "Forwarded"} {
					require.Equal(hc.headers.Values(name), actual.Header.Values(name), name)
				}
			})
		}
	}
}
