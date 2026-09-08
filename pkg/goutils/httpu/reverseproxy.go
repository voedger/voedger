/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package httpu

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"slices"
	"strings"
)

// RestoreForwardedHeaders is a ReverseProxy.Rewrite callback that preserves
// incoming forwarding headers and appends the client IP to X-Forwarded-For.
// Incoming values named by Connection stay excluded, as with a Director callback.
func RestoreForwardedHeaders(req *httputil.ProxyRequest) {
	// With Director, Go appends the client IP even if the callback is empty.
	// With Rewrite, Go first strips all four forwarding headers from Out.
	// Restore incoming values: SetXForwarded alone would lose the existing
	// X-Forwarded-For chain and leave Forwarded unset.
	for _, name := range []string{"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "Forwarded"} {
		if connectionListsHeader(req.In.Header, name) {
			continue
		}
		if values, ok := req.In.Header[name]; ok {
			req.Out.Header[name] = slices.Clone(values)
		}
	}
	// Append the peer IP without SetXForwarded: even after restoring the chain,
	// that method overwrites X-Forwarded-Host from In.Host and X-Forwarded-Proto
	// from In.TLS. Director preserved supplied values and left absent ones unset.
	clientIP, _, err := net.SplitHostPort(req.In.RemoteAddr)
	if err != nil {
		return
	}
	prior, ok := req.Out.Header["X-Forwarded-For"]
	if ok && prior == nil {
		return // A nil value explicitly suppresses X-Forwarded-For.
	}
	if len(prior) > 0 {
		clientIP = strings.Join(prior, ", ") + ", " + clientIP
	}
	req.Out.Header.Set("X-Forwarded-For", clientIP)
}

func connectionListsHeader(header http.Header, name string) bool {
	for _, value := range header.Values("Connection") {
		for token := range strings.SplitSeq(value, ",") {
			if textproto.CanonicalMIMEHeaderKey(textproto.TrimString(token)) == name {
				return true
			}
		}
	}
	return false
}
