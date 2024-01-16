/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/builtin"
	coreutils "github.com/voedger/voedger/pkg/utils"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

type ServiceFactory func(commandsChannel CommandChannel, partitionID istructs.PartitionID) pipeline.IService
type CommandChannel iprocbus.ServiceChannel
type OperatorSyncActualizer pipeline.ISyncOperator
type SyncActualizerFactory func(vvmCtx context.Context, partitionID istructs.PartitionID) pipeline.ISyncOperator
type VVMName string

type ValidateFunc func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID) (err error)

type ICommandMessage interface {
	Body() []byte
	AppQName() istructs.AppQName
	WSID() istructs.WSID // url WSID
	Sender() ibus.ISender
	PartitionID() istructs.PartitionID
	RequestCtx() context.Context
	Command() appdef.ICommand
	Token() string
	Host() string
}

type xPath string

type commandProcessorMetrics struct {
	vvm     string
	app     istructs.AppQName
	metrics imetrics.IMetrics
}

func (m *commandProcessorMetrics) increase(metricName string, valueDelta float64) {
	m.metrics.IncreaseApp(metricName, m.vvm, m.app, valueDelta)
}

type cmdWorkpiece struct {
	asp                          istructs.IAppStructsProvider
	appStructs                   istructs.IAppStructs
	requestData                  coreutils.MapObject
	cmdMes                       ICommandMessage
	argsObject                   istructs.IObject
	unloggedArgsObject           istructs.IObject
	reb                          istructs.IRawEventBuilder
	rawEvent                     istructs.IRawEvent
	pLogEvent                    istructs.IPLogEvent
	err                          error
	workspace                    *workspace
	idGenerator                  *implIDGenerator
	eca                          istructs.ExecCommandArgs
	metrics                      commandProcessorMetrics
	syncProjectorsStart          time.Time
	principals                   []iauthnz.Principal
	principalPayload             payloads.PrincipalPayload
	parsedCUDs                   []parsedCUD
	wsDesc                       istructs.IRecord
	hostStateProvider            *hostStateProvider
	wsInitialized                bool
	cmdResultBuilder             istructs.IObjectBuilder
	cmdResult                    istructs.IObject
	resources                    istructs.IResources
	cmdFunc                      istructs.ICommandFunction
	appPartitionRestartScheduled bool
}

type implIDGenerator struct {
	istructs.IIDGenerator
	generatedIDs map[istructs.RecordID]istructs.RecordID
}

type parsedCUD struct {
	opKind         iauthnz.OperationKindType
	existingRecord istructs.IRecord // create -> nil
	id             int64
	qName          appdef.QName
	fields         coreutils.MapObject
	xPath          xPath
}

type implICommandMessage struct {
	body        []byte
	appQName    istructs.AppQName // need to determine where to send c.sys.Init request on create a new workspace
	wsid        istructs.WSID
	sender      ibus.ISender
	partitionID istructs.PartitionID
	requestCtx  context.Context
	command     appdef.ICommand
	token       string
	host        string
}

type wrongArgsCatcher struct {
	pipeline.NOOP
}

type hostStateProvider struct {
	as               istructs.IAppStructs
	cud              istructs.ICUD
	wsid             istructs.WSID
	principals       []iauthnz.Principal
	state            state.IHostState
	token            string
	cmdResultBuilder istructs.IObjectBuilder
}

func newHostStateProvider(ctx context.Context, pid istructs.PartitionID, secretReader isecrets.ISecretReader) *hostStateProvider {
	p := &hostStateProvider{}
	p.state = state.ProvideCommandProcessorStateFactory()(ctx, p.getAppStructs, state.SimplePartitionIDFunc(pid), p.getWSID, secretReader, p.getCUD, p.getPrincipals, p.getToken, builtin.MaxCUDs, p.getCmdResultBuilder)
	return p
}

func (p *hostStateProvider) getAppStructs() istructs.IAppStructs { return p.as }
func (p *hostStateProvider) getWSID() istructs.WSID              { return p.wsid }
func (p *hostStateProvider) getCUD() istructs.ICUD               { return p.cud }
func (p *hostStateProvider) getPrincipals() []iauthnz.Principal {
	return p.principals
}
func (p *hostStateProvider) getToken() string                             { return p.token }
func (p *hostStateProvider) getCmdResultBuilder() istructs.IObjectBuilder { return p.cmdResultBuilder }
func (p *hostStateProvider) get(appStructs istructs.IAppStructs, wsid istructs.WSID, cud istructs.ICUD, principals []iauthnz.Principal, token string, cmdResultBuilder istructs.IObjectBuilder) state.IHostState {
	p.as = appStructs
	p.wsid = wsid
	p.cud = cud
	p.principals = principals
	p.token = token
	p.cmdResultBuilder = cmdResultBuilder
	return p.state
}
