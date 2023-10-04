/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ibus "github.com/untillpro/airs-ibus"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/exp/maps"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	workspacemgmt "github.com/voedger/voedger/pkg/sys/authnz/workspace"
	"github.com/voedger/voedger/pkg/sys/blobber"
	"github.com/voedger/voedger/pkg/sys/builtin"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func (cm *implICommandMessage) Body() []byte                      { return cm.body }
func (cm *implICommandMessage) AppQName() istructs.AppQName       { return cm.appQName }
func (cm *implICommandMessage) WSID() istructs.WSID               { return cm.wsid }
func (cm *implICommandMessage) Sender() interface{}               { return cm.sender }
func (cm *implICommandMessage) PartitionID() istructs.PartitionID { return cm.partitionID }
func (cm *implICommandMessage) RequestCtx() context.Context       { return cm.requestCtx }
func (cm *implICommandMessage) Resource() istructs.IResource      { return cm.resource }
func (cm *implICommandMessage) Token() string                     { return cm.token }
func (cm *implICommandMessage) Host() string                      { return cm.host }

func NewCommandMessage(requestCtx context.Context, body []byte, appQName istructs.AppQName, wsid istructs.WSID, sender interface{},
	partitionID istructs.PartitionID, resource istructs.IResource, token string, host string) ICommandMessage {
	return &implICommandMessage{
		body:        body,
		appQName:    appQName,
		wsid:        wsid,
		sender:      sender,
		partitionID: partitionID,
		requestCtx:  requestCtx,
		resource:    resource,
		token:       token,
		host:        host,
	}
}

// need for collection.ProvideSyncActualizer()
func (c *cmdWorkpiece) AppDef() appdef.IAppDef {
	return c.appStructs.AppDef()
}

// need for collection.ProvideSyncActualizer(), q.sys.EnrichPrincipalToken, c.sys.ChangePassword
func (c *cmdWorkpiece) AppQName() istructs.AppQName {
	return c.cmdMes.AppQName()
}

// need for sync projectors which are using wsid.GetNextWSID()
func (c *cmdWorkpiece) Context() context.Context {
	return c.cmdMes.RequestCtx()
}

// need for collection.ProvideSyncActualizer()
func (c *cmdWorkpiece) Event() istructs.IPLogEvent {
	return c.pLogEvent
}

// used by ProvideSyncActualizerFactory
func (c *cmdWorkpiece) WSID() istructs.WSID {
	return c.cmdMes.WSID()
}

func (ap *appPartition) getWorkspace(wsid istructs.WSID) *workspace {
	ws, ok := ap.workspaces[wsid]
	if !ok {
		ws = &workspace{
			NextWLogOffset: istructs.FirstOffset,
			idGenerator:    istructsmem.NewIDGenerator(),
		}
		ap.workspaces[wsid] = ws
	}
	return ws
}

func (cmdProc *cmdProc) getAppPartition(ctx context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.cmdMes.AppQName()
	ap, ok := cmdProc.appPartitions[cmd.cmdMes.AppQName()]
	if !ok {
		if ap, err = cmdProc.recovery(ctx, cmd); err != nil {
			return err
		}
		cmdProc.appPartitions[cmd.cmdMes.AppQName()] = ap
	}
	cmdProc.appPartition = ap
	return nil
}

func (cmdProc *cmdProc) getCmdResultBuilder(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	res := cmd.cmdFunc.ResultType()
	if res != appdef.NullQName {
		cfg := cmdProc.cfgs[cmd.cmdMes.AppQName()]
		cmd.cmdResultBuilder = istructsmem.NewIObjectBuilder(cfg, res)
	}
	return nil
}

func (cmdProc *cmdProc) buildCommandArgs(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	hs := cmd.hostStateProvider.get(cmd.appStructs, cmd.cmdMes.WSID(), cmd.reb.CUDBuilder(), cmd.principals, cmd.cmdMes.Token(), cmd.cmdResultBuilder)
	hs.ClearIntents()
	cmd.eca = istructs.ExecCommandArgs{
		CommandPrepareArgs: istructs.CommandPrepareArgs{
			PrepareArgs: istructs.PrepareArgs{
				ArgumentObject: cmd.argsObject,
				Workspace:      cmd.cmdMes.WSID(),
				Workpiece:      work,
			},
			ArgumentUnloggedObject: cmd.unloggedArgsObject,
		},
		State:   hs,
		Intents: hs,
	}
	return
}

func (cmdProc *cmdProc) recovery(ctx context.Context, cmd *cmdWorkpiece) (*appPartition, error) {
	ap := &appPartition{
		workspaces:     map[istructs.WSID]*workspace{},
		nextPLogOffset: istructs.FirstOffset,
	}
	cb := func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		ws := ap.getWorkspace(event.Workspace())
		_ = event.CUDs(func(rec istructs.ICUDRow) error { // no errors to return
			if rec.IsNew() {
				t := cmd.AppDef().Type(rec.QName())
				ws.idGenerator.UpdateOnSync(rec.ID(), t)
			}
			return nil
		})
		ws.NextWLogOffset = event.WLogOffset() + 1
		ap.nextPLogOffset = plogOffset + 1
		return nil
	}

	if err := cmd.appStructs.Events().ReadPLog(ctx, cmdProc.pNumber, istructs.FirstOffset, istructs.ReadToTheEnd, cb); err != nil {
		return nil, err
	}
	worskapcesJSON, err := json.Marshal(ap.workspaces)
	if err != nil {
		// error impossible
		// notest
		return nil, err
	}
	logger.Info(fmt.Sprintf(`app "%s" partition %d recovered: nextPLogOffset %d, workspaces: %s`, cmd.cmdMes.AppQName(), cmdProc.pNumber, ap.nextPLogOffset, string(worskapcesJSON)))
	return ap, nil
}

func getIDGenerator(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.idGenerator = &implIDGenerator{
		IIDGenerator: cmd.workspace.idGenerator,
		generatedIDs: map[istructs.RecordID]istructs.RecordID{},
	}
	return nil
}

func (cmdProc *cmdProc) putPLog(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.pLogEvent, err = cmd.appStructs.Events().PutPlog(cmd.rawEvent, nil, cmd.idGenerator)
	cmdProc.appPartition.nextPLogOffset++
	return
}

func getWSDesc(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if !coreutils.IsDummyWS(cmd.cmdMes.WSID()) {
		cmd.wsDesc, err = cmd.appStructs.Records().GetSingleton(cmd.cmdMes.WSID(), authnz.QNameCDocWorkspaceDescriptor)
	}
	return
}

func checkWSInitialized(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	wsDesc := work.(*cmdWorkpiece).wsDesc
	funcQName := cmd.cmdMes.Resource().(istructs.ICommandFunction).QName()
	if coreutils.IsDummyWS(cmd.cmdMes.WSID()) {
		return nil
	}
	if funcQName == workspacemgmt.QNameCommandCreateWorkspace ||
		funcQName == workspacemgmt.QNameCommandCreateWorkspaceID || // happens on creating a child of an another workspace
		funcQName == builtin.QNameCommandInit {
		return nil
	}
	if wsDesc.QName() != appdef.NullQName {
		if funcQName == blobber.QNameCommandUploadBLOBHelper {
			return nil
		}
		if wsDesc.AsInt64(workspacemgmt.Field_InitCompletedAtMs) > 0 && len(wsDesc.AsString(workspacemgmt.Field_InitError)) == 0 {
			cmd.wsInitialized = true
			return nil
		}
		if funcQName == istructs.QNameCommandCUD {
			if iauthnz.IsSystemPrincipal(cmd.principals, cmd.cmdMes.WSID()) {
				// system -> allow any CUD to upload template, see https://github.com/voedger/voedger/issues/648
				return nil
			}
		}
	}
	return errWSNotInited
}

func checkWSActive(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if coreutils.IsDummyWS(cmd.cmdMes.WSID()) {
		return nil
	}
	if iauthnz.IsSystemPrincipal(cmd.principals, cmd.cmdMes.WSID()) {
		// system -> allow to work in any case
		return nil
	}
	if cmd.wsDesc.QName() == appdef.NullQName {
		return nil
	}
	if cmd.wsDesc.AsInt32(authnz.Field_Status) == int32(authnz.WorkspaceStatus_Active) {
		return nil
	}
	return processors.ErrWSInactive
}

func getAppStructs(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.appStructs, err = cmd.asp.AppStructs(cmd.cmdMes.AppQName())
	return
}

func limitCallRate(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.appStructs.IsFunctionRateLimitsExceeded(cmd.cmdMes.Resource().QName(), cmd.cmdMes.WSID()) {
		return coreutils.NewHTTPErrorf(http.StatusTooManyRequests)
	}
	return nil
}

func (cmdProc *cmdProc) authenticate(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	req := iauthnz.AuthnRequest{
		Host:        cmd.cmdMes.Host(),
		RequestWSID: cmd.cmdMes.WSID(),
		Token:       cmd.cmdMes.Token(),
	}
	if cmd.principals, cmd.principalPayload, err = cmdProc.authenticator.Authenticate(cmd.cmdMes.RequestCtx(), cmd.appStructs,
		cmd.appStructs.AppTokens(), req); err != nil {
		return coreutils.NewHTTPError(http.StatusUnauthorized, err)
	}
	return
}

func (cmdProc *cmdProc) authorizeRequest(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	req := iauthnz.AuthzRequest{
		OperationKind: iauthnz.OperationKind_EXECUTE,
		Resource:      cmd.cmdMes.Resource().QName(),
	}
	ok, err := cmdProc.authorizer.Authorize(cmd.appStructs, cmd.principals, req)
	if err != nil {
		return err
	}
	if !ok {
		return coreutils.NewHTTPErrorf(http.StatusForbidden)
	}
	return nil
}

func getFunction(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.cmdFunc = cmd.cmdMes.Resource().(istructs.ICommandFunction)
	return
}

func unmarshalRequestBody(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.cmdFunc.ParamsType() == istructs.QNameJSON {
		cmd.requestData["args"] = map[string]interface{}{
			processors.Field_JSONDef_Body: string(cmd.cmdMes.Body()),
		}
	} else if err = json.Unmarshal(cmd.cmdMes.Body(), &cmd.requestData); err != nil {
		err = fmt.Errorf("failed to unmarshal request body: %w", err)
	}
	return
}

func (cmdProc *cmdProc) getWorkspace(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.workspace = cmdProc.appPartition.getWorkspace(cmd.cmdMes.WSID())
	return nil
}

func (cmdProc *cmdProc) getRawEventBuilder(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	grebp := istructs.GenericRawEventBuilderParams{
		HandlingPartition: cmd.cmdMes.PartitionID(),
		Workspace:         cmd.cmdMes.WSID(),
		QName:             cmd.cmdFunc.QName(),
		RegisteredAt:      istructs.UnixMilli(cmdProc.now().UnixMilli()),
		PLogOffset:        cmdProc.appPartition.nextPLogOffset,
		WLogOffset:        cmd.workspace.NextWLogOffset,
	}

	switch cmd.cmdMes.Resource().QName() {
	case builtin.QNameCommandInit: // kept to not to break existing events only
		cmd.reb = cmd.appStructs.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				SyncedAt:                     istructs.UnixMilli(cmdProc.now().UnixMilli()),
				GenericRawEventBuilderParams: grebp,
			},
		)
	default:
		cmd.reb = cmd.appStructs.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: grebp,
			},
		)
	}
	return nil
}

func getArgsObject(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.cmdFunc.ParamsType() == appdef.NullQName {
		return nil
	}
	aob := cmd.reb.ArgumentObjectBuilder()
	if argsIntf, exists := cmd.requestData["args"]; exists {
		args, ok := argsIntf.(map[string]interface{})
		if !ok {
			return errors.New(`"args" field must be an object`)
		}
		parsType := cmd.appStructs.AppDef().Type(cmd.cmdFunc.ParamsType())
		if err = istructsmem.FillElementFromJSON(args, parsType, aob); err != nil {
			return err
		}
	}
	if cmd.argsObject, err = aob.Build(); err != nil {
		err = fmt.Errorf("argument object build failed: %w", err)
	}
	return
}

func getUnloggedArgsObject(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.cmdFunc.UnloggedParamsType() == appdef.NullQName {
		return nil
	}
	auob := cmd.reb.ArgumentUnloggedObjectBuilder()
	if unloggedArgsIntf, exists := cmd.requestData["unloggedArgs"]; exists {
		unloggedArgs, ok := unloggedArgsIntf.(map[string]interface{})
		if !ok {
			return errors.New(`"unloggedArgs" field must be an object`)
		}
		unloggedParsType := cmd.appStructs.AppDef().Type(cmd.cmdFunc.UnloggedParamsType())
		if err = istructsmem.FillElementFromJSON(unloggedArgs, unloggedParsType, auob); err != nil {
			return err
		}
	}
	if cmd.unloggedArgsObject, err = auob.Build(); err != nil {
		err = fmt.Errorf("unlogged argument object build failed: %w", err)
	}
	return
}

func (xp xPath) Errorf(mes string, args ...interface{}) error {
	return fmt.Errorf(string(xp)+": "+mes, args...)
}

func (xp xPath) Error(err error) error {
	return xp.Errorf("%w", err)
}

func execCommand(_ context.Context, work interface{}) (err error) {
	begin := time.Now()
	defer work.(*cmdWorkpiece).metrics.increase(ExecSeconds, time.Since(begin).Seconds())
	cmd := work.(*cmdWorkpiece)
	return cmd.cmdFunc.Exec(cmd.eca)
}

func buildRawEvent(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.rawEvent, err = cmd.reb.BuildRawEvent()
	status := http.StatusBadRequest
	if errors.Is(err, istructsmem.ErrRecordIDUniqueViolation) {
		status = http.StatusConflict
	}
	err = coreutils.WrapSysError(err, status)
	return
}

func validateCmdResult(ctx context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.cmdResultBuilder != nil {
		cmdResult, err := cmd.cmdResultBuilder.Build()
		if err != nil {
			return err
		}
		cmd.cmdResult = cmdResult
	}
	return nil
}

func (cmdProc *cmdProc) validate(ctx context.Context, work interface{}) (err error) {
	defer func() {
		err = coreutils.WrapSysError(err, http.StatusForbidden)
	}()
	cmd := work.(*cmdWorkpiece)
	for _, appEventValidator := range cmd.appStructs.EventValidators() {
		if err = appEventValidator(ctx, cmd.rawEvent, cmd.appStructs, cmd.cmdMes.WSID()); err != nil {
			return
		}
	}
	for _, appCUDValidator := range cmd.appStructs.CUDValidators() {
		err = cmd.rawEvent.CUDs(func(rec istructs.ICUDRow) error {
			if istructs.ValidatorMatchByQName(appCUDValidator, rec.QName(), cmd.cmdMes.WSID(), cmd.cmdFunc.QName()) {
				if err := appCUDValidator.Validate(ctx, cmd.appStructs, rec, cmd.cmdMes.WSID(), cmd.cmdFunc.QName()); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return
		}
	}
	return nil
}

func parseCUDs(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cuds, _, err := cmd.requestData.AsObjects("cuds")
	if err != nil {
		return err
	}
	for cudNumber, cudIntf := range cuds {
		if cudNumber > builtin.MaxCUDs {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "too many cuds, max is", builtin.MaxCUDs)
		}
		xPath := xPath("cuds[" + strconv.Itoa(cudNumber) + "]")
		cudDataMap, ok := cudIntf.(map[string]interface{})
		if !ok {
			return xPath.Errorf("not an object")
		}
		cudData := coreutils.MapObject(cudDataMap)

		parsedCUD := parsedCUD{
			xPath: xPath,
		}

		parsedCUD.fields, ok, err = cudData.AsObject("fields")
		if err != nil {
			return xPath.Error(err)
		}
		if !ok {
			return xPath.Errorf(`"fields" missing`)
		}
		// sys.ID внутри -> create, снаружи -> update
		isCreate := false
		if parsedCUD.id, isCreate, err = parsedCUD.fields.AsInt64(appdef.SystemField_ID); err != nil {
			return xPath.Error(err)
		}
		if isCreate {
			parsedCUD.opKind = iauthnz.OperationKind_INSERT
			qNameStr, _, err := parsedCUD.fields.AsString(appdef.SystemField_QName)
			if err != nil {
				return xPath.Error(err)
			}
			if parsedCUD.qName, err = appdef.ParseQName(qNameStr); err != nil {
				return xPath.Error(err)
			}
		} else {
			parsedCUD.opKind = iauthnz.OperationKind_UPDATE
			if parsedCUD.id, ok, err = cudData.AsInt64(appdef.SystemField_ID); err != nil {
				return xPath.Error(err)
			}
			if !ok {
				return xPath.Errorf(`"sys.ID" missing`)
			}
			if parsedCUD.existingRecord, err = cmd.appStructs.Records().Get(cmd.cmdMes.WSID(), true, istructs.RecordID(parsedCUD.id)); err != nil {
				return
			}
			if parsedCUD.qName = parsedCUD.existingRecord.QName(); parsedCUD.qName == appdef.NullQName {
				return coreutils.NewHTTPError(http.StatusNotFound, xPath.Errorf("record with queried id %d does not exist", parsedCUD.id))
			}
		}
		cmd.parsedCUDs = append(cmd.parsedCUDs, parsedCUD)
	}
	return err
}

func checkArgsRefIntegrity(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.argsObject != nil {
		if err = builtin.CheckRefIntegrity(cmd.argsObject, cmd.appStructs, cmd.cmdMes.WSID()); err != nil {
			return err
		}
	}
	if cmd.unloggedArgsObject != nil {
		return builtin.CheckRefIntegrity(cmd.unloggedArgsObject, cmd.appStructs, cmd.cmdMes.WSID())
	}
	return nil
}

func (cmdProc *cmdProc) authorizeCUDs(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	for _, parsedCUD := range cmd.parsedCUDs {
		req := iauthnz.AuthzRequest{
			OperationKind: parsedCUD.opKind,
			Resource:      parsedCUD.qName,
			Fields:        maps.Keys(parsedCUD.fields),
		}
		ok, err := cmdProc.authorizer.Authorize(cmd.appStructs, cmd.principals, req)
		if err != nil {
			return parsedCUD.xPath.Error(err)
		}
		if !ok {
			return coreutils.NewHTTPError(http.StatusForbidden, parsedCUD.xPath.Errorf("operation forbidden"))
		}
	}
	return
}

func (cmdProc *cmdProc) writeCUDs(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	for _, parsedCUD := range cmd.parsedCUDs {
		var rowWriter istructs.IRowWriter
		if parsedCUD.opKind == iauthnz.OperationKind_INSERT {
			rowWriter = cmd.reb.CUDBuilder().Create(parsedCUD.qName)
			rowWriter.PutRecordID(appdef.SystemField_ID, istructs.RecordID(parsedCUD.id))
		} else {
			rowWriter = cmd.reb.CUDBuilder().Update(parsedCUD.existingRecord)
		}
		if err := coreutils.Marshal(rowWriter, parsedCUD.fields); err != nil {
			return parsedCUD.xPath.Error(err)
		}
	}
	return nil
}

func (osp *wrongArgsCatcher) OnErr(err error, _ interface{}, _ pipeline.IWorkpieceContext) (newErr error) {
	return coreutils.WrapSysError(err, http.StatusBadRequest)
}

func applyPLogEvent(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if err = cmd.appStructs.Records().Apply(cmd.pLogEvent); err == nil {
		// actually WLogOffset must be increased after successfult write to WLog.
		// but new WLogOffset is needed for next step - Sync Projectors
		// so increase WLogOffsets here - right before Sync Projectors
		cmd.workspace.NextWLogOffset++
	}
	return
}

func (cmdProc *cmdProc) n10n(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmdProc.n10nBroker.Update(in10n.ProjectionKey{
		App:        cmd.AppQName(),
		Projection: projectors.PlogQName,
		WS:         istructs.WSID(cmdProc.pNumber),
	}, cmd.rawEvent.PLogOffset())
	logger.Verbose("updated plog event on offset ", cmd.rawEvent.PLogOffset(), ", pnumber ", cmdProc.pNumber)
	return nil
}

func putWLog(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	err = cmd.appStructs.Events().PutWlog(cmd.pLogEvent)
	if err != nil {
		cmd.workspace.NextWLogOffset++
	}
	return
}

func syncProjectorsBegin(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.syncProjectorsStart = time.Now()
	return
}

func syncProjectorsEnd(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.metrics.increase(ProjectorsSeconds, time.Since(cmd.syncProjectorsStart).Seconds())
	cmd.syncProjectorsStart = time.Time{}
	return
}

type opSendResponse struct {
	pipeline.NOOP
	bus ibus.IBus
}

func (sr *opSendResponse) DoSync(_ context.Context, work interface{}) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.err != nil {
		cmd.metrics.increase(ErrorsTotal, 1.0)
		//if error occurred somewhere in syncProjectors we have to measure elapsed time
		if !cmd.syncProjectorsStart.IsZero() {
			cmd.metrics.increase(ProjectorsSeconds, time.Since(cmd.syncProjectorsStart).Seconds())
		}
		logger.Error(cmd.err)
		coreutils.ReplyErr(sr.bus, cmd.cmdMes.Sender(), cmd.err)
		return
	}
	body := bytes.NewBufferString(fmt.Sprintf(`{"CurrentWLogOffset":%d`, cmd.Event().WLogOffset()))
	if len(cmd.idGenerator.generatedIDs) > 0 {
		body.WriteString(`,"NewIDs":{`)
		for rawID, generatedID := range cmd.idGenerator.generatedIDs {
			body.WriteString(fmt.Sprintf(`"%d":%d,`, rawID, generatedID))
		}
		body.Truncate(body.Len() - 1)
		body.WriteString("}")
	}
	if cmd.cmdResult != nil {
		cmdResult := coreutils.ObjectToMap(cmd.cmdResult, cmd.AppDef())
		cmdResultBytes, err := json.Marshal(cmdResult)
		if err != nil {
			// notest
			return err
		}
		body.WriteString(`,"Result":`)
		body.WriteString(string(cmdResultBytes))
	}
	body.WriteString("}")
	coreutils.ReplyJSON(sr.bus, cmd.cmdMes.Sender(), http.StatusOK, body.String())
	return nil
}

// nolint (result is always nil)
func (sr *opSendResponse) OnErr(err error, work interface{}, _ pipeline.IWorkpieceContext) error {
	work.(*cmdWorkpiece).err = err
	return nil
}

func (idGen *implIDGenerator) NextID(rawID istructs.RecordID, t appdef.IType) (storageID istructs.RecordID, err error) {
	storageID, err = idGen.IIDGenerator.NextID(rawID, t)
	idGen.generatedIDs[rawID] = storageID
	return
}
