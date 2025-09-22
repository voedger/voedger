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

func New(vvmCtx context.Context, federationURL func() *url.URL, adminPortGetter func() int) (federation IFederation, cleanup func()) {
	httpClient, cln := httpu.NewIHTTPClient(
		httpu.WithSkipRetryOn503(),
		httpu.WithOptsValidator(httpu.DenyGETAndDiscardResponse), // to prevent discarding possible sys.Error
	)
	fed := &implIFederation{
		httpClient:      httpClient,
		federationURL:   federationURL,
		adminPortGetter: adminPortGetter,
		vvmCtx:          vvmCtx,
	}
	return fed, cln
}
