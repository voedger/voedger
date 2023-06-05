/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package workspace

import (
	"context"
	"fmt"
	"net/http"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/collection"
	"github.com/voedger/voedger/pkg/sys/invite"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideDeactivateWorkspace(cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, tokensAPI itokens.ITokens, federation coreutils.IFederation,
	asp istructs.IAppStructsProvider) {

	// c.sys.DeactivateWorkspace
	// target app, target WSID
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdInitiateDeactivateWorkspace,
		appdef.NullQName,
		appdef.NullQName,
		appdef.NullQName,
		cmdInitiateDeactivateWorkspaceExec,
	))

	// c.sys.OnWorkspaceDeactivated
	// owner app, owner WSID
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "OnWorkspaceDeactivated"),
		adf.AddObject(appdef.NewQName(appdef.SysPackage, "OnWorkspaceDeactivatedParams")).
			AddField(Field_OwnerWSID, appdef.DataKind_int64, true).
			AddField(sysshared.Field_WSName, appdef.DataKind_string, true).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		cmdOnWorkspaceDeactivatedExec,
	))

	// c.sys.OnJoinedWorkspaceDeactivated
	// target app, profile WSID
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "OnJoinedWorkspaceDeactivated"),
		adf.AddObject(appdef.NewQName(appdef.SysPackage, "OnJoinedWorkspaceDeactivatedParams")).
			AddField(field_InvitedToWSID, appdef.DataKind_int64, true).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		cmdOnJoinedWorkspaceDeactivateExec,
	))

	// c.sys.OnChildWorkspaceDeactivated
	// ownerApp/ownerWSID
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "OnChildWorkspaceDeactivated"),
		adf.AddObject(appdef.NewQName(appdef.SysPackage, "OnChildWorkspaceDeactivatedParams")).
			AddField(Field_OwnerID, appdef.DataKind_int64, true).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		cmdOnChildWorkspaceDeactivatedExec,
	))

	adf.AddObject(qNameProjectorApplyDeactivateWorkspace)

	// target app, target WSID
	cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameProjectorApplyDeactivateWorkspace,
			EventsFilter: []appdef.QName{qNameCmdInitiateDeactivateWorkspace},
			Func:         projectorApplyDeactivateWorkspace(federation, cfg.Name, tokensAPI, asp),
		}
	})
}

func cmdInitiateDeactivateWorkspaceExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	kb, err := args.State.KeyBuilder(state.RecordsStorage, sysshared.QNameCDocWorkspaceDescriptor)
	if err != nil {
		// notest
		return err
	}
	kb.PutQName(state.Field_Singleton, sysshared.QNameCDocWorkspaceDescriptor)
	wsDesc, err := args.State.MustExist(kb)
	if err != nil {
		// notest
		return err
	}
	status := wsDesc.AsInt32(sysshared.Field_Status)
	if status != int32(sysshared.WorkspaceStatus_Active) {
		return coreutils.NewHTTPErrorf(http.StatusConflict, "Workspace Status is not Active")
	}

	wsDescUpdater, err := args.Intents.UpdateValue(kb, wsDesc)
	if err != nil {
		// notest
		return err
	}
	wsDescUpdater.PutInt32(sysshared.Field_Status, int32(sysshared.WorkspaceStatus_ToBeDeactivated))
	return nil
}

func cmdOnJoinedWorkspaceDeactivateExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	invitedToWSID := args.ArgumentObject.AsInt64(field_InvitedToWSID)
	svCDocJoinedWorkspace, skb, ok, err := invite.GetCDocJoinedWorkspace(args.State, args.Intents, invitedToWSID)
	if err != nil || !ok {
		return err
	}
	if !svCDocJoinedWorkspace.AsBool(appdef.SystemField_IsActive) {
		return nil
	}
	cdocJoinedWorkspaceUpdater, err := args.Intents.UpdateValue(skb, svCDocJoinedWorkspace)
	if err != nil {
		// notest
		return err
	}
	cdocJoinedWorkspaceUpdater.PutBool(appdef.SystemField_IsActive, false)
	return nil
}

// app/pseudoProfileWSID, ownerApp
func cmdOnWorkspaceDeactivatedExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	ownerWSID := args.ArgumentObject.AsInt64(Field_OwnerWSID)
	wsName := args.ArgumentObject.AsString(sysshared.Field_WSName)
	kb, err := args.State.KeyBuilder(state.ViewRecordsStorage, QNameViewWorkspaceIDIdx)
	if err != nil {
		// notest
		return err
	}
	kb.PutInt64(Field_OwnerWSID, ownerWSID)
	kb.PutString(sysshared.Field_WSName, wsName)
	viewRec, ok, err := args.State.CanExist(kb)
	if err != nil {
		// notest
		return err
	}
	if !ok {
		logger.Verbose("workspace", wsName, ":", ownerWSID, "is not mentioned in view.sys.WorkspaceIDId")
		return
	}
	idOfCDocWorkspaceID := viewRec.AsRecordID(field_IDOfCDocWorkspaceID)
	kb, err = args.State.KeyBuilder(state.RecordsStorage, QNameCDocWorkspaceID)
	if err != nil {
		// notest
		return err
	}
	kb.PutRecordID(state.Field_ID, idOfCDocWorkspaceID)
	cdocWorkspaceID, err := args.State.MustExist(kb)
	if err != nil {
		// notest
		return err
	}

	if !cdocWorkspaceID.AsBool(appdef.SystemField_IsActive) {
		logger.Verbose("cdoc.sys.WorkspaceID is inactive already")
		return nil
	}

	cdocWorkspaceIDUpdater, err := args.Intents.UpdateValue(kb, cdocWorkspaceID)
	if err != nil {
		// notest
		return err
	}
	cdocWorkspaceIDUpdater.PutBool(appdef.SystemField_IsActive, false)
	return nil
}

// ownerApp/ownerWSID
func cmdOnChildWorkspaceDeactivatedExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	ownerID := args.ArgumentObject.AsInt64(Field_OwnerID)
	kb, err := args.State.KeyBuilder(state.RecordsStorage, appdef.NullQName)
	if err != nil {
		// notest
		return err
	}
	kb.PutRecordID(state.Field_ID, istructs.RecordID(ownerID))
	cdocOwnerSV, err := args.State.MustExist(kb)
	if err != nil {
		// notest
		return err
	}
	if !cdocOwnerSV.AsBool(appdef.SystemField_IsActive) {
		return nil
	}
	cdocOwnerUpdater, err := args.Intents.UpdateValue(kb, cdocOwnerSV)
	if err != nil {
		// notest
		return err
	}
	cdocOwnerUpdater.PutBool(appdef.SystemField_IsActive, false)
	return nil
}

// target app, target WSID
func projectorApplyDeactivateWorkspace(federation coreutils.IFederation, appQName istructs.AppQName, tokensAPI itokens.ITokens,
	asp istructs.IAppStructsProvider) func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		kb, err := s.KeyBuilder(state.RecordsStorage, sysshared.QNameCDocWorkspaceDescriptor)
		if err != nil {
			// notest
			return err
		}
		kb.PutQName(state.Field_Singleton, sysshared.QNameCDocWorkspaceDescriptor)
		wsDesc, err := s.MustExist(kb)
		if err != nil {
			// notest
			return err
		}
		ownerApp := wsDesc.AsString(Field_OwnerApp)
		ownerWSID := wsDesc.AsInt64(Field_OwnerWSID)
		ownerID := wsDesc.AsInt64(Field_OwnerID)

		sysToken, err := payloads.GetSystemPrincipalToken(tokensAPI, appQName)
		if err != nil {
			// notest
			return err
		}

		// Foreach cdoc.sys.Subject
		as, err := asp.AppStructs(appQName)
		if err != nil {
			// notest
			return err
		}
		subjectsKB := as.ViewRecords().KeyBuilder(collection.QNameViewCollection)
		subjectsKB.PutInt32(collection.Field_PartKey, collection.PartitionKeyCollection)
		subjectsKB.PutQName(collection.Field_DocQName, sysshared.QNameCDocSubject)
		err = as.ViewRecords().Read(context.Background(), event.Workspace(), subjectsKB, func(_ istructs.IKey, value istructs.IValue) (err error) {
			subject := value.AsRecord(collection.Field_Record)
			if istructs.SubjectKindType(subject.AsInt32(sysshared.Field_SubjectKind)) != istructs.SubjectKind_User {
				return nil
			}
			profileWSID := istructs.WSID(subject.AsInt64(sysshared.Field_ProfileWSID))
			// app is always current
			// impossible to have logins from different apps among subjects (Michael said)
			url := fmt.Sprintf(`api/%s/%d/c.sys.OnJoinedWorkspaceDeactivated`, appQName, profileWSID)

			body := fmt.Sprintf(`{"args":{"InvitedToWSID":%d}}`, event.Workspace())
			_, err = coreutils.FederationFunc(federation.URL(), url, body, coreutils.WithAuthorizeBy(sysToken), coreutils.WithDiscardResponse())
			return err
		})
		if err != nil {
			// notestdebt
			return err
		}

		// currentApp/ApplicationWS/c.sys.OnWorkspaceDeactivated(OnwerWSID, WSName)
		wsName := wsDesc.AsString(sysshared.Field_WSName)
		body := fmt.Sprintf(`{"args":{"OwnerWSID":%d, "WSName":"%s"}}`, ownerWSID, wsName)
		cdocWorkspaceIDWSID := coreutils.GetPseudoWSID(istructs.WSID(ownerWSID), wsName, event.Workspace().ClusterID())
		if _, err := coreutils.FederationFunc(federation.URL(), fmt.Sprintf("api/%s/%d/c.sys.OnWorkspaceDeactivated", ownerApp, cdocWorkspaceIDWSID), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken)); err != nil {
			return fmt.Errorf("c.sys.OnWorkspaceDeactivated failed: %w", err)
		}

		// c.sys.OnChildWorkspaceDeactivated(ownerID))
		body = fmt.Sprintf(`{"args":{"OwnerID":%d}}`, ownerID)
		if _, err := coreutils.FederationFunc(federation.URL(), fmt.Sprintf("api/%s/%d/c.sys.OnChildWorkspaceDeactivated", ownerApp, ownerWSID), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken)); err != nil {
			return fmt.Errorf("c.sys.OnChildWorkspaceDeactivated failed: %w", err)
		}

		// cdoc.sys.WorkspaceDescriptor.Status = Inactive
		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Status":%d}}]}`, wsDesc.AsRecordID(appdef.SystemField_ID), sysshared.WorkspaceStatus_Inactive)
		if _, err := coreutils.FederationFunc(federation.URL(), fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken)); err != nil {
			return fmt.Errorf("cdoc.sys.WorkspaceDescriptor.Status=Inactive failed: %w", err)
		}

		logger.Info("workspace", wsDesc.AsString(sysshared.Field_WSName), "deactivated")
		return nil
	}
}
