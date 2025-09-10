/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"context"
	"net"
	"net/http"
	"slices"
)

func NewIHTTPClient(defaultOpts ...ReqOptFunc) (client IHTTPClient, clenup func()) {
	// set linger - see https://github.com/voedger/voedger/issues/415
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		err = conn.(*net.TCPConn).SetLinger(0)
		return conn, err
	}
	client = &implIHTTPClient{
		client:      &http.Client{Transport: tr},
		defaultOpts: append(slices.Clone(constDefaultOpts), defaultOpts...),
	}
	return client, client.CloseIdleConnections
}
