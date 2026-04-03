/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package processors

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

type VVMName string

type APIPath int

type IProcessorWorkpiece interface {
	pipeline.IWorkpiece
	AppPartitions() appparts.IAppPartitions
	AppPartition() appparts.IAppPartition
	GetPrincipals() []iauthnz.Principal
	Roles() []appdef.QName
	ResetRateLimit(appdef.QName, appdef.OperationKind)
	LogCtx() context.Context
}

type IProjectorWorkpiece interface {
	pipeline.IWorkpiece
	AppPartition() appparts.IAppPartition
	Event() istructs.IPLogEvent
	LogCtx() context.Context
	PLogOffset() istructs.Offset
}
