/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"net/url"

	"github.com/voedger/voedger/pkg/coreutils"
)

func New(federationURL func() *url.URL, adminPortGetter func() int) (federation IFederation, cleanup func()) {
	httpClient, cln := coreutils.NewIHTTPClient()
	fed := &implIFederation{
		httpClient:      httpClient,
		federationURL:   federationURL,
		adminPortGetter: adminPortGetter,
	}
	return fed, cln
}

func NewForQP(federation IFederation) IFederationForQP {
	return &implIFederationForQP{
		fed: federation,
	}
}
