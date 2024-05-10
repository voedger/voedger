/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"net/url"

	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/blobber"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type IFederation interface {
	// POST(relativeURL string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	// GET(relativeURL string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	Func(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.FuncResponse, error)
	UploadBLOB(appQName istructs.AppQName, wsid istructs.WSID, blobName string, blobMimeType string, blobContent []byte,
		optFuncs ...coreutils.ReqOptFunc) (blobID istructs.RecordID, err error)
	UploadBLOBs(appQName istructs.AppQName, wsid istructs.WSID, blobs []blobber.BLOB, optFuncs ...coreutils.ReqOptFunc) (blobIDs []istructs.RecordID, err error)
	ReadBLOB(appQName istructs.AppQName, wsid istructs.WSID, blobID istructs.RecordID, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error)
	URLStr() string
	Port() int
	N10NUpdate(key in10n.ProjectionKey, val int64, optFuncs ...coreutils.ReqOptFunc) error
	N10NSubscribe(projectionKey in10n.ProjectionKey) (offsetsChan OffsetsChan, unsubscribe func(), err error)
	AdminFunc(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.FuncResponse, error)
}

type implIFederation struct {
	httpClient      coreutils.IHTTPClient
	federationURL   func() *url.URL
	adminPortGetter func() int
}

type OffsetsChan chan istructs.Offset
