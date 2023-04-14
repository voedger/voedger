/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"context"
	"time"

	"github.com/untillpro/voedger/pkg/iauthnz"
	"github.com/untillpro/voedger/pkg/iprocbus"
	"github.com/untillpro/voedger/pkg/isecrets"
	"github.com/untillpro/voedger/pkg/istructs"
	payloads "github.com/untillpro/voedger/pkg/itokens-payloads"
	imetrics "github.com/untillpro/voedger/pkg/metrics"
	"github.com/untillpro/voedger/pkg/pipeline"
	"github.com/untillpro/voedger/pkg/state"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

type ServiceFactory func(commandsChannel CommandChannel, partitionID istructs.PartitionID) pipeline.IService
type CommandChannel iprocbus.ServiceChannel
type OperatorSyncActualizer pipeline.ISyncOperator
type SyncActualizerFactory func(hvmCtx context.Context, partitionID istructs.PartitionID) pipeline.ISyncOperator
type HVMName string

type ValidateFunc func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID) (err error)

type ICommandMessage interface {
	Body() []byte
	AppQName() istructs.AppQName
	WSID() istructs.WSID // url WSID
	Sender() interface{}
	PartitionID() istructs.PartitionID
	RequestCtx() context.Context
	Resource() istructs.IResource
	Token() string
	Host() string
}

type xPath string

type commandProcessorMetrics struct {
	hvm     string
	app     istructs.AppQName
	metrics imetrics.IMetrics
}

func (m *commandProcessorMetrics) increase(metricName string, valueDelta float64) {
	m.metrics.IncreaseApp(metricName, m.hvm, m.app, valueDelta)
}

type cmdWorkpiece struct {
	asp                 istructs.IAppStructsProvider
	appStructs          istructs.IAppStructs
	requestData         coreutils.MapObject
	cmdMes              ICommandMessage
	cmdFunc             istructs.ICommandFunction
	argsObject          istructs.IObject
	unloggedArgsObject  istructs.IObject
	reb                 istructs.IRawEventBuilder
	rawEvent            istructs.IRawEvent
	pLogEvent           istructs.IPLogEvent
	wLogEvent           istructs.IWLogEvent
	err                 error
	workspace           *workspace
	generatedIDs        map[istructs.RecordID]istructs.RecordID
	eca                 istructs.ExecCommandArgs
	metrics             commandProcessorMetrics
	syncProjectorsStart time.Time
	principals          []iauthnz.Principal
	principalPayload    payloads.PrincipalPayload
	parsedCUDs          []parsedCUD
	wsDesc              istructs.IRecord
	checkWSDescUpdating bool
	hostStateProvider   *hostStateProvider
	refIntegrityToCheck map[string]istructs.RecordID
}

type parsedCUD struct {
	opKind         iauthnz.OperationKindType
	existingRecord istructs.IRecord // create -> nil
	id             int64
	qName          istructs.QName
	fields         coreutils.MapObject
	xPath          xPath
}

type implICommandMessage struct {
	body        []byte
	appQName    istructs.AppQName // need to determine where to send c.sys.Init request on create a new workspace
	wsid        istructs.WSID
	sender      interface{}
	partitionID istructs.PartitionID
	requestCtx  context.Context
	resource    istructs.IResource
	token       string
	host        string
}

type wrongArgsCatcher struct {
	pipeline.NOOP
}

type hostStateProvider struct {
	as         istructs.IAppStructs
	cud        istructs.ICUD
	wsid       istructs.WSID
	principals []iauthnz.Principal
	state      state.IHostState
	token      string
}

func newHostStateProvider(ctx context.Context, pid istructs.PartitionID, secretReader isecrets.ISecretReader) *hostStateProvider {
	p := &hostStateProvider{}
	p.state = state.ProvideCommandProcessorStateFactory()(ctx, p.getAppStructs, state.SimplePartitionIDFunc(pid), p.getWSID, secretReader, p.getCUD, p.getPrincipals, p.getToken, intentsLimit)
	return p
}

func (p *hostStateProvider) getAppStructs() istructs.IAppStructs { return p.as }
func (p *hostStateProvider) getWSID() istructs.WSID              { return p.wsid }
func (p *hostStateProvider) getCUD() istructs.ICUD               { return p.cud }
func (p *hostStateProvider) getPrincipals() []iauthnz.Principal {
	return p.principals
}
func (p *hostStateProvider) getToken() string { return p.token }
func (p *hostStateProvider) get(appStructs istructs.IAppStructs, wsid istructs.WSID, cud istructs.ICUD, principals []iauthnz.Principal, token string) state.IHostState {
	p.as = appStructs
	p.wsid = wsid
	p.cud = cud
	p.principals = principals
	p.token = token
	return p.state
}
