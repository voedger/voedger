/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/utils/federation"
)

// template recordID -> template fieldName -> uploaded blobID to set to fieldName
type blobsMap map[int64]map[string]int64

type WSPostInitFunc func(targetAppQName appdef.AppQName, wsKind appdef.QName, newWSID istructs.WSID, federation federation.IFederation, authToken string) (err error)
