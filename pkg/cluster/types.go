/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
)

type update struct {
	setFields     map[string]interface{}
	kind          updateKind
	qName         appdef.QName
	key           map[string]interface{}
	wsid          istructs.WSID
	id            istructs.RecordID
	partitionID   istructs.PartitionID
	offset        istructs.Offset
	appQName      istructs.AppQName
	sql           string
	appStructs    istructs.IAppStructs
	appParts      appparts.IAppPartitions
	qNameTypeKind appdef.TypeKind
}

type updateKind int
