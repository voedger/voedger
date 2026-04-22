/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/processors/actualizers"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/stateprovide"
)

type ServiceFactory func(commandsChannel CommandChannel) pipeline.IService
type CommandChannel iprocbus.ServiceChannel
type OperatorSyncActualizer pipeline.ISyncOperator
type SyncActualizerFactory func(vvmCtx context.Context, partitionID istructs.PartitionID) pipeline.ISyncOperator

type ValidateFunc func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID) (err error)

type ICommandMessage interface {
	Body() []byte
	AppQName() appdef.AppQName
	WSID() istructs.WSID // url WSID
	Responder() bus.IResponder
	PartitionID() istructs.PartitionID
	RequestCtx() context.Context
	QName() appdef.QName // APIv1 -> cmd QName, APIv2 -> cmdQName or DocQName
	Token() string
	Host() string
	APIPath() processors.APIPath
	DocID() istructs.RecordID
	Method() string
	Origin() string
}

type xPath string

type commandProcessorMetrics struct {
	vvmName string
	app     appdef.AppQName
	metrics imetrics.IMetrics
}

func (m *commandProcessorMetrics) increase(metricName string, valueDelta float64) {
	m.metrics.IncreaseApp(metricName, m.vvmName, m.app, valueDelta)
}

type cmdWorkpiece struct {
	appParts                     appparts.IAppPartitions
	appPart                      appparts.IAppPartition
	appStructs                   istructs.IAppStructs
	requestData                  coreutils.MapObject
	cmdMes                       ICommandMessage
	argsObject                   istructs.IObject
	unloggedArgsObject           istructs.IObject
	reb                          istructs.IRawEventBuilder
	rawEvent                     istructs.IRawEvent
	pLogEvent                    istructs.IPLogEvent
	appPartition                 *appPartition
	workspace                    *workspace
	idGeneratorReporter          *implIDGeneratorReporter
	eca                          istructs.ExecCommandArgs
	metrics                      commandProcessorMetrics
	syncProjectorsStart          time.Time
	principals                   []iauthnz.Principal
	roles                        []appdef.QName
	parsedCUDs                   []parsedCUD
	wsDesc                       istructs.IRecord
	hostState                    *reusableHostState
	wsInitialized                bool
	cmdResultBuilder             istructs.IObjectBuilder
	cmdResult                    istructs.IObject
	iCommand                     appdef.ICommand
	iWorkspace                   appdef.IWorkspace
	appPartitionRestartScheduled bool
	cmdQName                     appdef.QName
	statusCodeOfSuccess          int
	reapplier                    istructs.IEventReapplier
	commandCtxStorage            istructs.IStateValue
	cmdResToLog                  string
	pLogOffset                   istructs.Offset // need for logging
	logCtx                       context.Context // enriched log ctx from logEventAndCUDs (woffset, poffset, evqname)
}

var _ processors.IProcessorWorkpiece = (*cmdWorkpiece)(nil)

type implIDGeneratorReporter struct {
	istructs.IIDGenerator
	generatedIDs map[istructs.RecordID]istructs.RecordID
}

type parsedCUD struct {
	opKind         appdef.OperationKind // update can not be activate\deactivate because IsActive modified -> other fields update is not allowed, see
	existingRecord istructs.IRecord     // create -> nil
	id             int64
	qName          appdef.QName
	fields         coreutils.MapObject
	xPath          xPath
}

type implICommandMessage struct {
	body        []byte
	appQName    appdef.AppQName // need to determine where to send c.sys.Init request on create a new workspace
	wsid        istructs.WSID
	responder   bus.IResponder
	partitionID istructs.PartitionID
	requestCtx  context.Context
	qName       appdef.QName // APIv1 -> cmd QName, APIv2 -> cmdQName or DocQName
	token       string
	host        string
	apiPath     processors.APIPath
	docID       istructs.RecordID
	method      string
	origin      string
}

type wrongArgsCatcher struct {
	pipeline.NOOP
}

type reusableHostState struct {
	wp    *cmdWorkpiece
	state state.IHostState
}

func newReusableHostState(ctx context.Context, secretReader isecrets.ISecretReader) *reusableHostState {
	b := &reusableHostState{}
	b.state = stateprovide.ProvideCommandProcessorStateFactory()(ctx,
		func() istructs.IAppStructs { return b.wp.appStructs },
		func() istructs.PartitionID { return b.wp.cmdMes.PartitionID() },
		func() istructs.WSID { return b.wp.cmdMes.WSID() },
		secretReader,
		func() istructs.ICUD { return b.wp.reb.CUDBuilder() },
		func() []iauthnz.Principal { return b.wp.principals },
		func() string { return b.wp.cmdMes.Token() },
		actualizers.DefaultIntentsLimit,
		func() istructs.IObjectBuilder { return b.wp.cmdResultBuilder },
		func() istructs.CommandPrepareArgs { return b.wp.eca.CommandPrepareArgs },
		func() istructs.IObject { return b.wp.argsObject },
		func() istructs.IObject { return b.wp.unloggedArgsObject },
		func() istructs.Offset { return b.wp.workspace.NextWLogOffset },
		state.NullOpts,
		func() string { return b.wp.cmdMes.Origin() },
	)
	return b
}

func (b *reusableHostState) bind(wp *cmdWorkpiece) {
	b.wp = wp
}
