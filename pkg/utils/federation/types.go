/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"net/url"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type implIFederation struct {
	httpClient      coreutils.IHTTPClient
	federationURL   func() *url.URL
	adminPortGetter func() int
}

type OffsetsChan chan istructs.Offset
