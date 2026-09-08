/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package httpu

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestoreForwardedHeaders(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		headers    http.Header
		expected   http.Header
	}{
		{
			name:       "no incoming forwarding headers",
			remoteAddr: "192.0.2.1:1234",
			headers:    nil,
			expected:   http.Header{"X-Forwarded-For": {"192.0.2.1"}},
		},
		{
			name:       "preserve incoming forwarding headers",
			remoteAddr: "192.0.2.1:1234",
			headers: http.Header{
				"X-Forwarded-For":   {"198.51.100.2"},
				"X-Forwarded-Host":  {"original.example"},
				"X-Forwarded-Proto": {"https"},
				"Forwarded":         {"for=198.51.100.2;proto=https"},
			},
			expected: http.Header{
				"X-Forwarded-For":   {"198.51.100.2, 192.0.2.1"},
				"X-Forwarded-Host":  {"original.example"},
				"X-Forwarded-Proto": {"https"},
				"Forwarded":         {"for=198.51.100.2;proto=https"},
			},
		},
		{
			name:       "fold multiple values and append IPv6 client",
			remoteAddr: "[2001:db8::1]:1234",
			headers: http.Header{
				"X-Forwarded-For":   {"192.0.2.1", "198.51.100.2, 203.0.113.3"},
				"X-Forwarded-Host":  {"original.example", "proxy.example"},
				"X-Forwarded-Proto": {"https", "http"},
				"Forwarded":         {"for=192.0.2.1;proto=https", "for=198.51.100.2"},
			},
			expected: http.Header{
				"X-Forwarded-For":   {"192.0.2.1, 198.51.100.2, 203.0.113.3, 2001:db8::1"},
				"X-Forwarded-Host":  {"original.example", "proxy.example"},
				"X-Forwarded-Proto": {"https", "http"},
				"Forwarded":         {"for=192.0.2.1;proto=https", "for=198.51.100.2"},
			},
		},
		{
			name:       "nil header suppresses client IP",
			remoteAddr: "192.0.2.1:1234",
			headers:    http.Header{"X-Forwarded-For": nil},
			expected:   http.Header{"X-Forwarded-For": nil},
		},
		{
			name:       "empty header allows client IP",
			remoteAddr: "192.0.2.1:1234",
			headers:    http.Header{"X-Forwarded-For": {}},
			expected:   http.Header{"X-Forwarded-For": {"192.0.2.1"}},
		},
		{
			name:       "invalid remote address preserves incoming header",
			remoteAddr: "invalid",
			headers:    http.Header{"X-Forwarded-For": {"198.51.100.2"}},
			expected:   http.Header{"X-Forwarded-For": {"198.51.100.2"}},
		},
		{
			name:       "invalid remote address without incoming header",
			remoteAddr: "invalid",
			headers:    nil,
			expected:   http.Header{},
		},
		{
			name:       "Connection tokens exclude incoming forwarding values",
			remoteAddr: "192.0.2.1:1234",
			headers: http.Header{
				"Connection":        {" x-forwarded-for, X-Forwarded-HOST ", "\tFORWARDED, x-FoRwArDeD-pRoTo , X-Test-Hop"},
				"X-Forwarded-For":   {"198.51.100.2"},
				"X-Forwarded-Host":  {"original.example"},
				"X-Forwarded-Proto": {"https"},
				"Forwarded":         {"for=198.51.100.2"},
				"X-Test-Hop":        {"remove"},
			},
			expected: http.Header{"X-Forwarded-For": {"192.0.2.1"}},
		},
		{
			name:       "Connection excludes only nominated forwarding headers",
			remoteAddr: "192.0.2.1:1234",
			headers: http.Header{
				"Connection":       {"X-Forwarded-Host"},
				"X-Forwarded-Host": {"original.example"},
				"X-Forwarded-For":  {"198.51.100.2"},
				"Forwarded":        {"for=198.51.100.2"},
			},
			expected: http.Header{
				"X-Forwarded-For": {"198.51.100.2, 192.0.2.1"},
				"Forwarded":       {"for=198.51.100.2"},
			},
		},
		{
			name:       "Connection removal takes precedence over nil suppression",
			remoteAddr: "192.0.2.1:1234",
			headers:    http.Header{"Connection": {"X-Forwarded-For"}, "X-Forwarded-For": nil},
			expected:   http.Header{"X-Forwarded-For": {"192.0.2.1"}},
		},
		{
			name:       "invalid peer with Connection nominated header",
			remoteAddr: "invalid",
			headers:    http.Header{"Connection": {"X-Forwarded-For"}, "X-Forwarded-For": {"198.51.100.2"}},
			expected:   http.Header{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			in := httptest.NewRequest(http.MethodGet, "http://upstream.example/path?query=value", http.NoBody)
			in.RemoteAddr = tc.remoteAddr
			maps.Copy(in.Header, tc.headers.Clone())
			in.Header.Set("Keep-Alive", "timeout=5")
			in.Header.Set("X-End-To-End", "keep")
			original := in.Clone(t.Context())
			// Capture after ReverseProxy has applied its own header cleanup.
			transport := make(forwardedHeadersTransport, 1)
			proxy := httputil.ReverseProxy{Rewrite: RestoreForwardedHeaders, Transport: transport}
			response := httptest.NewRecorder()

			proxy.ServeHTTP(response, in)

			require.Equal(http.StatusOK, response.Code)
			require.Len(transport, 1)
			out := <-transport
			// Comparing the complete header map also catches leaked hop-by-hop headers.
			expected := http.Header{"X-End-To-End": {"keep"}, "User-Agent": {""}}
			maps.Copy(expected, tc.expected)
			require.Equal(expected, out.Header)
			require.Equal(original.URL, out.URL)
			require.Equal(original.Host, out.Host)
			require.Equal(original.Header, in.Header)
			for _, values := range out.Header {
				if len(values) > 0 {
					values[0] = "changed"
				}
			}
			require.Equal(original.Header, in.Header, "outbound headers must not alias inbound headers")
		})
	}
}

type forwardedHeadersTransport chan *http.Request

func (tr forwardedHeadersTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr <- req
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}, nil
}
