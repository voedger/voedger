/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"io"
	"net/url"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type implIFederation struct {
	httpClient      coreutils.IHTTPClient
	federationURL   func() *url.URL
	adminPortGetter func() int
}

type OffsetsChan chan istructs.Offset

// for read and write
// caller must read out and close the reader
type BLOBReader struct {
	io.ReadCloser
	iblobstorage.DescrType
}
