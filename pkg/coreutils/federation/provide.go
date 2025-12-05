/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"context"
	"net/url"

	"github.com/voedger/voedger/pkg/goutils/httpu"
)

func New(vvmCtx context.Context, federationURL func() *url.URL, adminPortGetter func() int, policyForWithRetry PolicyOptsForWithRetry) (federation IFederation, cleanup func()) {
	httpClient, cln := httpu.NewIHTTPClient(
		httpu.WithNoRetryPolicy(),
		httpu.WithOptsValidator(httpu.DenyGETAndDiscardResponse), // to prevent discarding possible sys.Error
	)
	fed := &implIFederation{
		httpClient:             httpClient,
		federationURL:          federationURL,
		adminPortGetter:        adminPortGetter,
		vvmCtx:                 vvmCtx,
		policyOptsForWithRetry: policyForWithRetry,
	}
	return fed, cln
}
