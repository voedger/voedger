/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

type iFederationBase interface {
	Func(relativeURL string, body string, optFuncs ...httpu.ReqOptFunc) (*FuncResponse, error)
	Query(relativeURL string, optFuncs ...httpu.ReqOptFunc) (*FuncResponse, error)
	UploadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobReader iblobstorage.BLOBReader, optFuncs ...httpu.ReqOptFunc) (blobID istructs.RecordID, err error)
	UploadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobReader iblobstorage.BLOBReader, duration iblobstorage.DurationType,
		optFuncs ...httpu.ReqOptFunc) (blobSUUID iblobstorage.SUUID, err error)
	ReadBLOB(appQName appdef.AppQName, wsid istructs.WSID, ownerRecord appdef.QName, ownerRecordField appdef.FieldName, ownerID istructs.RecordID,
		optFuncs ...httpu.ReqOptFunc) (iblobstorage.BLOBReader, error)
	ReadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobSUUID iblobstorage.SUUID, optFuncs ...httpu.ReqOptFunc) (iblobstorage.BLOBReader, error)
	URLStr() string
	Port() int
	N10NUpdate(key in10n.ProjectionKey, val int64, optFuncs ...httpu.ReqOptFunc) error
	N10NSubscribe(projectionKey in10n.ProjectionKey) (offsetsChan OffsetsChan, unsubscribe func(), err error)
	AdminFunc(relativeURL string, body string, optFuncs ...httpu.ReqOptFunc) (*FuncResponse, error)
}

type IFederation interface {
	iFederationBase
	WithRetry() IFederationWithRetry
}

// need for Workspace init workflow
// has WithRetryOn503 default option
type IFederationWithRetry interface {
	iFederationBase
	dummy()
}
