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
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/processors/actualizers"
	"golang.org/x/exp/maps"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/blobber"
	"github.com/voedger/voedger/pkg/sys/builtin"
	workspacemgmt "github.com/voedger/voedger/pkg/sys/workspace"
)

func (cm *implICommandMessage) Body() []byte                      { return cm.body }
func (cm *implICommandMessage) AppQName() appdef.AppQName         { return cm.appQName }
func (cm *implICommandMessage) WSID() istructs.WSID               { return cm.wsid }
func (cm *implICommandMessage) Responder() bus.IResponder         { return cm.responder }
func (cm *implICommandMessage) PartitionID() istructs.PartitionID { return cm.partitionID }
func (cm *implICommandMessage) RequestCtx() context.Context       { return cm.requestCtx }
func (cm *implICommandMessage) QName() appdef.QName               { return cm.qName }
func (cm *implICommandMessage) Token() string                     { return cm.token }
func (cm *implICommandMessage) Host() string                      { return cm.host }

func NewCommandMessage(requestCtx context.Context, body []byte, appQName appdef.AppQName, wsid istructs.WSID,
	responder bus.IResponder, partitionID istructs.PartitionID, qName appdef.QName, token string, host string) ICommandMessage {
	return &implICommandMessage{
		body:        body,
		appQName:    appQName,
		wsid:        wsid,
		responder:   responder,
		partitionID: partitionID,
		requestCtx:  requestCtx,
		qName:       qName,
		token:       token,
		host:        host,
	}
}

// used in projectors.newSyncBranch()
func (c *cmdWorkpiece) AppPartition() appparts.IAppPartition {
	return c.appPart
}

// used in c.cluster.VSqlUpdate to determinate partitionID by WSID
// used in c.registry.CreateLogin to dtermine if the target app is deployed
func (c *cmdWorkpiece) AppPartitions() appparts.IAppPartitions {
	return c.appParts
}

// need for sync projectors which are using wsid.GetNextWSID()
func (c *cmdWorkpiece) Context() context.Context {
	return c.cmdMes.RequestCtx()
}

// used in projectors.NewSyncActualizerFactoryFactory
func (c *cmdWorkpiece) Event() istructs.IPLogEvent {
	return c.pLogEvent
}

// need for update corrupted in c.cluster.VSqlUpdate and for various funcs of sys package
func (c *cmdWorkpiece) GetAppStructs() istructs.IAppStructs {
	return c.appStructs
}

// https://github.com/voedger/voedger/issues/3163
func (c *cmdWorkpiece) GetUserPrincipalName() string {
	for _, prn := range c.principals {
		if prn.Kind == iauthnz.PrincipalKind_User {
			return prn.Name
		}
	}
	return ""
}

// borrows app partition for command
func (c *cmdWorkpiece) borrow() (err error) {
	if c.appPart, err = c.appParts.Borrow(c.cmdMes.AppQName(), c.cmdMes.PartitionID(), appparts.ProcessorKind_Command); err != nil {
		if errors.Is(err, appparts.ErrNotFound) || errors.Is(err, appparts.ErrNotAvailableEngines) { // partition is not deployed yet -> ErrNotFound
			return coreutils.NewHTTPError(http.StatusServiceUnavailable, err)
		}
		// notest
		return err
	}
	c.appStructs = c.appPart.AppStructs()
	return nil
}

// releases resources:
//   - borrowed app partition
//   - plog event
func (c *cmdWorkpiece) Release() {
	if ev := c.pLogEvent; ev != nil {
		c.pLogEvent = nil
		ev.Release()
	}
	if ap := c.appPart; ap != nil {
		c.appStructs = nil
		c.appPart = nil
		ap.Release()
	}
}

func borrowAppPart(_ context.Context, work pipeline.IWorkpiece) error {
	return work.(*cmdWorkpiece).borrow()
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

func (cmdProc *cmdProc) getAppPartition(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	appPartitions, ok := cmdProc.appsPartitions[cmd.cmdMes.AppQName()]
	if !ok {
		appPartitions = map[istructs.PartitionID]*appPartition{}
		cmdProc.appsPartitions[cmd.cmdMes.AppQName()] = appPartitions
	}
	appPartition, ok := appPartitions[cmd.cmdMes.PartitionID()]
	if !ok {
		if appPartition, err = cmdProc.recovery(ctx, cmd); err != nil {
			return fmt.Errorf("partition %d recovery failed: %w", cmd.cmdMes.PartitionID(), err)
		}
		appPartitions[cmd.cmdMes.PartitionID()] = appPartition
	}
	cmd.appPartition = appPartition
	return nil
}

func getIWorkspace(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)

	switch cmd.cmdMes.QName() {
	case workspacemgmt.QNameCommandCreateWorkspace:
		// cmd.iWorkspace should be nil
	default:
		ws := cmd.wsDesc.AsQName(authnz.Field_WSKind)
		if cmd.iWorkspace = cmd.appStructs.AppDef().WorkspaceByDescriptor(ws); cmd.iWorkspace == nil {
			panic(fmt.Errorf("workspace %s does not exist", ws))
		}
	}

	return nil
}

func getICommand(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	var cmdType appdef.IType
	if cmd.iWorkspace == nil {
		// DummyWS or c.sys.CreateWorkspace
		cmdType = cmd.appStructs.AppDef().Type(cmd.cmdMes.QName())
	} else {
		if cmdType = cmd.iWorkspace.Type(cmd.cmdMes.QName()); cmdType.Kind() == appdef.TypeKind_null {
			return fmt.Errorf("command %s does not exist in workspace %s", cmd.cmdMes.QName(), cmd.iWorkspace.QName())
		}
	}
	ok := false
	cmd.iCommand, ok = cmdType.(appdef.ICommand)
	if !ok {
		return fmt.Errorf("%s is not a command", cmd.cmdMes.QName())
	}
	return nil
}

func (cmdProc *cmdProc) getCmdResultBuilder(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmdResultType := cmd.iCommand.Result()
	if cmdResultType != nil {
		cmd.cmdResultBuilder = cmd.appStructs.ObjectBuilder(cmdResultType.QName())
	}
	return nil
}

func (cmdProc *cmdProc) buildCommandArgs(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.eca.CommandPrepareArgs = istructs.CommandPrepareArgs{
		PrepareArgs: istructs.PrepareArgs{
			ArgumentObject: cmd.argsObject,
			WSID:           cmd.cmdMes.WSID(),
			Workpiece:      work,
			Workspace:      cmd.iWorkspace,
		},
		ArgumentUnloggedObject: cmd.unloggedArgsObject,
	}

	hs := cmd.hostStateProvider.get(cmd.appStructs, cmd.cmdMes.WSID(), cmd.reb.CUDBuilder(),
		cmd.principals, cmd.cmdMes.Token(), cmd.cmdResultBuilder, cmd.eca.CommandPrepareArgs, cmd.workspace.NextWLogOffset,
		cmd.argsObject, cmd.unloggedArgsObject, cmd.cmdMes.PartitionID())
	hs.ClearIntents()

	cmd.eca.State = hs
	cmd.eca.Intents = hs

	return
}

func updateIDGeneratorFromO(root istructs.IObject, findType appdef.FindType, idGen istructs.IIDGenerator) {
	// new IDs only here because update is not allowed for ODocs in Args
	idGen.UpdateOnSync(root.AsRecordID(appdef.SystemField_ID), findType(root.QName()))
	for container := range root.Containers {
		// order of containers here is the order in the schema
		// but order in the request could be different
		// that is not a problem because for ODocs/ORecords ID generator will bump next ID only if syncID is actually next
		for c := range root.Children(container) {
			updateIDGeneratorFromO(c, findType, idGen)
		}
	}
}

func (cmdProc *cmdProc) recovery(ctx context.Context, cmd *cmdWorkpiece) (*appPartition, error) {
	ap := &appPartition{
		workspaces:     map[istructs.WSID]*workspace{},
		nextPLogOffset: istructs.FirstOffset,
	}
	var lastPLogEvent istructs.IPLogEvent
	cb := func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		ws := ap.getWorkspace(event.Workspace())

		for rec := range event.CUDs {
			if rec.IsNew() {
				t := cmd.appStructs.AppDef().Type(rec.QName())
				ws.idGenerator.UpdateOnSync(rec.ID(), t)
			}
		}
		ao := event.ArgumentObject()
		if cmd.appStructs.AppDef().Type(ao.QName()).Kind() == appdef.TypeKind_ODoc {
			updateIDGeneratorFromO(ao, cmd.appStructs.AppDef().Type, ws.idGenerator)
		}
		ws.NextWLogOffset = event.WLogOffset() + 1
		ap.nextPLogOffset = plogOffset + 1
		if lastPLogEvent != nil {
			lastPLogEvent.Release() // TODO: eliminate if there will be a better solution, see https://github.com/voedger/voedger/issues/1348
		}
		lastPLogEvent = event
		return nil
	}

	if err := cmd.appStructs.Events().ReadPLog(ctx, cmd.cmdMes.PartitionID(), istructs.FirstOffset, istructs.ReadToTheEnd, cb); err != nil {
		return nil, err
	}

	if lastPLogEvent != nil {
		// re-apply the last event
		cmd.pLogEvent = lastPLogEvent
		cmd.workspace = ap.getWorkspace(lastPLogEvent.Workspace())
		cmd.workspace.NextWLogOffset-- // cmdProc.storeOp will bump it
		if err := cmdProc.storeOp.DoSync(ctx, cmd); err != nil {
			return nil, err
		}
		cmd.pLogEvent = nil
		cmd.workspace = nil
		lastPLogEvent.Release() // TODO: eliminate if there will be a better solution, see https://github.com/voedger/voedger/issues/1348
	}

	worskapcesJSON, err := json.Marshal(ap.workspaces)
	if err != nil {
		// notest
		return nil, err
	}
	logger.Info(fmt.Sprintf(`app "%s" partition %d recovered: nextPLogOffset %d, workspaces: %s`, cmd.cmdMes.AppQName(), cmd.cmdMes.PartitionID(),
		ap.nextPLogOffset, string(worskapcesJSON)))
	return ap, nil
}

func getIDGenerator(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.idGenerator = &implIDGenerator{
		IIDGenerator: cmd.workspace.idGenerator,
		generatedIDs: map[istructs.RecordID]istructs.RecordID{},
	}
	return nil
}

func (cmdProc *cmdProc) putPLog(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.pLogEvent, err = cmd.appStructs.Events().PutPlog(cmd.rawEvent, nil, cmd.idGenerator); err != nil {
		cmd.appPartitionRestartScheduled = true
	} else {
		cmd.appPartition.nextPLogOffset++
	}
	return
}

func getWSDesc(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.wsDesc, err = cmd.appStructs.Records().GetSingleton(cmd.cmdMes.WSID(), authnz.QNameCDocWorkspaceDescriptor)
	return err
}

func checkWSInitialized(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	wsDesc := work.(*cmdWorkpiece).wsDesc
	cmdQName := cmd.cmdMes.QName()
	if cmdQName == workspacemgmt.QNameCommandCreateWorkspace || cmdQName == builtin.QNameCommandInit {
		return nil
	}
	if wsDesc.QName() != appdef.NullQName {
		if cmdQName == blobber.QNameCommandUploadBLOBHelper {
			return nil
		}
		if wsDesc.AsInt64(workspacemgmt.Field_InitCompletedAtMs) > 0 && len(wsDesc.AsString(workspacemgmt.Field_InitError)) == 0 {
			cmd.wsInitialized = true
			return nil
		}
		if cmdQName == istructs.QNameCommandCUD {
			if iauthnz.IsSystemPrincipal(cmd.principals, cmd.cmdMes.WSID()) {
				// system -> allow any CUD to upload template, see https://github.com/voedger/voedger/issues/648
				return nil
			}
		}
	}
	return processors.ErrWSNotInited
}

func checkWSActive(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
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

func limitCallRate(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.appStructs.IsFunctionRateLimitsExceeded(cmd.cmdMes.QName(), cmd.cmdMes.WSID()) {
		return coreutils.NewHTTPErrorf(http.StatusTooManyRequests)
	}
	return nil
}

func (cmdProc *cmdProc) authenticate(_ context.Context, work pipeline.IWorkpiece) (err error) {
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

func getPrincipalsRoles(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	for _, prn := range cmd.principals {
		if prn.Kind != iauthnz.PrincipalKind_Role {
			continue
		}
		cmd.roles = append(cmd.roles, prn.QName)
	}
	return nil
}

func (cmdProc *cmdProc) authorizeRequest(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)

	ws := cmd.iWorkspace
	if ws == nil {
		// dummy or c.sys.CreateWorkspace
		ws = cmd.iCommand.Workspace()
	}

	ok, err := cmd.appPart.IsOperationAllowed(ws, appdef.OperationKind_Execute, cmd.cmdMes.QName(), nil, cmd.roles)
	if err != nil {
		return err
	}
	if !ok {
		return coreutils.NewHTTPErrorf(http.StatusForbidden)
	}
	return nil
}

func roleNotFound(err error) bool {
	return errors.Is(err, appdef.ErrNotFoundError) && strings.Contains(err.Error(), "role")
}

func unmarshalRequestBody(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.iCommand.Param() != nil && cmd.iCommand.Param().QName() == istructs.QNameRaw {
		cmd.requestData["args"] = map[string]interface{}{
			processors.Field_RawObject_Body: string(cmd.cmdMes.Body()),
		}
	} else if err = coreutils.JSONUnmarshal(cmd.cmdMes.Body(), &cmd.requestData); err != nil {
		err = fmt.Errorf("failed to unmarshal request body: %w\n%s", err, cmd.cmdMes.Body())
	}
	return
}

func (cmdProc *cmdProc) getWorkspace(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.workspace = cmd.appPartition.getWorkspace(cmd.cmdMes.WSID())
	return nil
}

func (cmdProc *cmdProc) getRawEventBuilder(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	grebp := istructs.GenericRawEventBuilderParams{
		HandlingPartition: cmd.cmdMes.PartitionID(),
		Workspace:         cmd.cmdMes.WSID(),
		QName:             cmd.cmdMes.QName(),
		RegisteredAt:      istructs.UnixMilli(cmdProc.time.Now().UnixMilli()),
		PLogOffset:        cmd.appPartition.nextPLogOffset,
		WLogOffset:        cmd.workspace.NextWLogOffset,
	}

	switch cmd.cmdMes.QName() {
	case builtin.QNameCommandInit: // nolint, kept to not to break existing events only
		cmd.reb = cmd.appStructs.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				SyncedAt:                     istructs.UnixMilli(cmdProc.time.Now().UnixMilli()),
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

func getArgsObject(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.iCommand.Param() == nil {
		return nil
	}
	aob := cmd.reb.ArgumentObjectBuilder()
	args, exists, err := cmd.requestData.AsObject("args")
	if err != nil {
		return err
	}
	if exists {
		aob.FillFromJSON(args)
	}
	if cmd.argsObject, err = aob.Build(); err != nil {
		err = fmt.Errorf("argument object build failed: %w", err)
	}
	return
}

func getUnloggedArgsObject(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.iCommand.UnloggedParam() == nil {
		return nil
	}
	auob := cmd.reb.ArgumentUnloggedObjectBuilder()
	unloggedArgs, exists, err := cmd.requestData.AsObject("unloggedArgs")
	if err != nil {
		return err
	}
	if exists {
		auob.FillFromJSON(unloggedArgs)
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

func execCommand(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	begin := time.Now()

	err = cmd.appPart.Invoke(ctx, cmd.cmdMes.QName(), cmd.eca.State, cmd.eca.Intents)

	cmd.metrics.increase(ExecSeconds, time.Since(begin).Seconds())
	return err
}

func checkResponseIntent(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	return processors.CheckResponseIntent(cmd.hostStateProvider.state)
}

func buildRawEvent(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmd.rawEvent, err = cmd.reb.BuildRawEvent()
	status := http.StatusBadRequest
	if errors.Is(err, istructsmem.ErrRecordIDUniqueViolationError) {
		status = http.StatusConflict
	}
	err = coreutils.WrapSysError(err, status)
	return
}

func validateCmdResult(ctx context.Context, work pipeline.IWorkpiece) (err error) {
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

func (cmdProc *cmdProc) eventValidators(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	for _, appEventValidator := range cmd.appStructs.EventValidators() {
		if err = appEventValidator(ctx, cmd.rawEvent, cmd.appStructs, cmd.cmdMes.WSID()); err != nil {
			return coreutils.WrapSysError(err, http.StatusForbidden)
		}
	}
	return nil
}

func (cmdProc *cmdProc) cudsValidators(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	for _, appCUDValidator := range cmd.appStructs.CUDValidators() {
		for rec := range cmd.rawEvent.CUDs {
			if appCUDValidator.Match(rec, cmd.cmdMes.WSID(), cmd.cmdMes.QName()) {
				if err := appCUDValidator.Validate(ctx, cmd.appStructs, rec, cmd.cmdMes.WSID(), cmd.cmdMes.QName()); err != nil {
					return coreutils.WrapSysError(err, http.StatusForbidden)
				}
			}
		}
	}
	return nil
}

func (cmdProc *cmdProc) validateCUDsQNames(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	if cmd.iWorkspace == nil {
		// dummy or c.sys.CreateWorkspace
		return nil
	}
	for cud := range cmd.rawEvent.CUDs {
		if cmd.iWorkspace.Type(cud.QName()) == appdef.NullType {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("doc %s mentioned in resulting CUDs does not exist in the workspace %s",
				cud.QName(), cmd.wsDesc.AsQName(authnz.Field_WSKind)))
		}
	}
	return nil
}

func parseCUDs(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cuds, _, err := cmd.requestData.AsObjects("cuds")
	if err != nil {
		return err
	}
	if len(cuds) > builtin.MaxCUDs {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, "too many cuds: ", len(cuds), " is in the request, max is ", builtin.MaxCUDs)
	}
	for cudNumber, cudIntf := range cuds {
		cudXPath := xPath("cuds[" + strconv.Itoa(cudNumber) + "]")
		cudDataMap, ok := cudIntf.(map[string]interface{})
		if !ok {
			return cudXPath.Errorf("not an object")
		}
		cudData := coreutils.MapObject(cudDataMap)

		parsedCUD := parsedCUD{}

		parsedCUD.fields, ok, err = cudData.AsObject("fields")
		if err != nil {
			return cudXPath.Error(err)
		}
		if !ok {
			return cudXPath.Errorf(`"fields" missing`)
		}
		// sys.ID inside -> create, outside -> update
		isCreate := false
		if parsedCUD.id, isCreate, err = parsedCUD.fields.AsInt64(appdef.SystemField_ID); err != nil {
			return cudXPath.Error(err)
		}
		if parsedCUD.id < 0 {
			// TODO: cover by tests
			return cudXPath.Errorf("id can not be negative")
		}
		if isCreate {
			parsedCUD.opKind = appdef.OperationKind_Insert
			qNameStr, _, err := parsedCUD.fields.AsString(appdef.SystemField_QName)
			if err != nil {
				return cudXPath.Error(err)
			}
			if parsedCUD.qName, err = appdef.ParseQName(qNameStr); err != nil {
				return cudXPath.Error(fmt.Errorf("failed to parse sys.QName: %w", err))
			}
		} else {
			if parsedCUD.id, ok, err = cudData.AsInt64(appdef.SystemField_ID); err != nil {
				return cudXPath.Error(err)
			}
			if !ok {
				return cudXPath.Errorf(`"sys.ID" missing`)
			}
			if parsedCUD.existingRecord, err = cmd.appStructs.Records().Get(cmd.cmdMes.WSID(), true, istructs.RecordID(parsedCUD.id)); err != nil { // nolint G115
				return
			}
			if parsedCUD.qName = parsedCUD.existingRecord.QName(); parsedCUD.qName == appdef.NullQName {
				return coreutils.NewHTTPError(http.StatusNotFound, cudXPath.Errorf("record with queried id %d does not exist", parsedCUD.id))
			}
			// check for activate\deactivate
			providedIsActiveVal, isActiveModifying, err := parsedCUD.fields.AsBoolean(appdef.SystemField_IsActive)
			if err != nil {
				return err
			}
			// sys.IsActive is modifying -> other fields are not allowed, see [checkIsActiveInCUDs]
			if isActiveModifying {
				parsedCUD.opKind = appdef.OperationKind_Deactivate
				if providedIsActiveVal {
					parsedCUD.opKind = appdef.OperationKind_Activate
				}
			} else {
				parsedCUD.opKind = appdef.OperationKind_Update
			}
		}

		parsedCUD.xPath = xPath(fmt.Sprintf("%s %s %s", cudXPath, opKindDesc[parsedCUD.opKind], parsedCUD.qName))

		cmd.parsedCUDs = append(cmd.parsedCUDs, parsedCUD)
	}
	return err
}

func checkCUDsAllowedInCUDCmdOnly(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	if len(cmd.parsedCUDs) > 0 && cmd.cmdMes.QName() != istructs.QNameCommandCUD && cmd.cmdMes.QName() != builtin.QNameCommandInit {
		return errors.New("CUDs allowed for c.sys.CUD command only")
	}
	return nil
}

func checkArgsRefIntegrity(_ context.Context, work pipeline.IWorkpiece) (err error) {
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

// not a validator due of https://github.com/voedger/voedger/issues/1125
func checkIsActiveInCUDs(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	for _, cud := range cmd.parsedCUDs {
		if cud.opKind != appdef.OperationKind_Update && cud.opKind != appdef.OperationKind_Activate && cud.opKind != appdef.OperationKind_Deactivate {
			continue
		}
		hasOnlySystemFields := true
		sysIsActiveUpdating := false
		isActiveAndOtherFieldsMixedOnUpdate := false
		for fieldName := range cud.fields {
			if !appdef.IsSysField(fieldName) {
				hasOnlySystemFields = false
			} else if fieldName == appdef.SystemField_IsActive {
				sysIsActiveUpdating = true
			}
			if isActiveAndOtherFieldsMixedOnUpdate = sysIsActiveUpdating && !hasOnlySystemFields; isActiveAndOtherFieldsMixedOnUpdate {
				break
			}
		}
		if isActiveAndOtherFieldsMixedOnUpdate {
			return coreutils.NewHTTPError(http.StatusForbidden, errors.New("updating other fields is not allowed if sys.IsActive is updating"))
		}
	}
	return nil
}

func (cmdProc *cmdProc) authorizeRequestCUDs(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)

	ws := cmd.iWorkspace
	if ws == nil {
		// dummy or c.sys.CreateWorkspace
		ws = cmd.iCommand.Workspace()
	}
	for _, parsedCUD := range cmd.parsedCUDs {
		fields := maps.Keys(parsedCUD.fields)
		ok, err := cmd.appPart.IsOperationAllowed(ws, parsedCUD.opKind, parsedCUD.qName, fields, cmd.roles)
		if err != nil {
			if errors.Is(err, appdef.ErrNotFoundError) {
				err = coreutils.WrapSysError(err, http.StatusBadRequest)
			}
			return parsedCUD.xPath.Error(err)
		}
		if !ok {
			return coreutils.NewHTTPError(http.StatusForbidden, parsedCUD.xPath.Errorf("operation forbidden"))
		}
	}
	return
}

func (cmdProc *cmdProc) writeCUDs(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	for _, parsedCUD := range cmd.parsedCUDs {
		var cud istructs.IRowWriter
		if parsedCUD.opKind == appdef.OperationKind_Insert {
			cud = cmd.reb.CUDBuilder().Create(parsedCUD.qName)
			cud.PutRecordID(appdef.SystemField_ID, istructs.RecordID(parsedCUD.id)) // nolint G115
		} else {
			cud = cmd.reb.CUDBuilder().Update(parsedCUD.existingRecord)
		}
		if err := coreutils.MapToObject(parsedCUD.fields, cud); err != nil {
			return parsedCUD.xPath.Error(err)
		}
	}
	return nil
}

func (osp *wrongArgsCatcher) OnErr(err error, _ interface{}, _ pipeline.IWorkpieceContext) (newErr error) {
	return coreutils.WrapSysError(err, http.StatusBadRequest)
}

func (cmdProc *cmdProc) notifyAsyncActualizers(_ context.Context, work pipeline.IWorkpiece) (err error) {
	cmd := work.(*cmdWorkpiece)
	cmdProc.n10nBroker.Update(in10n.ProjectionKey{
		App:        cmd.cmdMes.AppQName(),
		Projection: actualizers.PLogUpdatesQName,
		WS:         istructs.WSID(cmd.cmdMes.PartitionID()),
	}, cmd.rawEvent.PLogOffset())
	logger.Verbose("async actualizers are notified: offset ", cmd.rawEvent.PLogOffset(), ", pnumber ", cmd.cmdMes.PartitionID())
	return nil
}

func sendResponse(cmd *cmdWorkpiece, handlingError error) {
	if handlingError != nil {
		cmd.metrics.increase(ErrorsTotal, 1.0)
		//if error occurred somewhere in syncProjectors we have to measure elapsed time
		if !cmd.syncProjectorsStart.IsZero() {
			cmd.metrics.increase(ProjectorsSeconds, time.Since(cmd.syncProjectorsStart).Seconds())
		}
		bus.ReplyErr(cmd.cmdMes.Responder(), handlingError)
		return
	}
	body := bytes.NewBufferString(fmt.Sprintf(`{"CurrentWLogOffset":%d`, cmd.pLogEvent.WLogOffset()))
	if len(cmd.idGenerator.generatedIDs) > 0 {
		body.WriteString(`,"NewIDs":{`)
		for rawID, generatedID := range cmd.idGenerator.generatedIDs {
			body.WriteString(fmt.Sprintf(`"%d":%d,`, rawID, generatedID))
		}
		body.Truncate(body.Len() - 1)
		body.WriteString("}")
		if logger.IsVerbose() {
			logger.Verbose("generated IDs:", cmd.idGenerator.generatedIDs)
		}
	}
	if cmd.cmdResult != nil {
		cmdResult := coreutils.ObjectToMap(cmd.cmdResult, cmd.appStructs.AppDef())
		cmdResultBytes, err := json.Marshal(cmdResult)
		if err != nil {
			// notest
			logger.Error("failed to marshal response: " + err.Error())
			return
		}
		body.WriteString(`,"Result":`)
		body.Write(cmdResultBytes)
	}
	body.WriteString("}")
	bus.ReplyJSON(cmd.cmdMes.Responder(), http.StatusOK, body.String())
}

func (idGen *implIDGenerator) NextID(rawID istructs.RecordID, t appdef.IType) (storageID istructs.RecordID, err error) {
	storageID, err = idGen.IIDGenerator.NextID(rawID, t)
	idGen.generatedIDs[rawID] = storageID
	return
}
