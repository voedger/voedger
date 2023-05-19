/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package workspace

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

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

func provideDeactivateWorkspace(cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, tokensAPI itokens.ITokens, federationURL func() *url.URL,
	asp istructs.IAppStructsProvider) {

	// c.sys.DeactivateWorkspace
	// target app, target WSID
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdDeactivateWorkspace,
		appdef.NullQName,
		appdef.NullQName,
		appdef.NullQName,
		cmdDeactivateWorkspaceExec,
	))

	// c.sys.OnWorkspaceDeactivated
	// owner app, owner WSID
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "OnWorkspaceDeactivated"),
		adf.AddStruct(appdef.NewQName(appdef.SysPackage, "OnWorkspaceDeactivatedParams"), appdef.DefKind_Object).
			AddField(Field_OwnerID, appdef.DataKind_int64, true).QName(),
		appdef.NullQName,
		appdef.NullQName,
		cmdOnWorkspaceDeactivatedExec,
	))

	// c.sys.OnJoinedWorkspaceDeactivated
	// target app, profile WSID
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "OnJoinedWorkspaceDeactivated"),
		adf.AddStruct(appdef.NewQName(appdef.SysPackage, "OnJoinedWorkspaceDeactivatedParams"), appdef.DefKind_Object).
			AddField(field_InvitedToWSID, appdef.DataKind_int64, true).QName(),
		appdef.NullQName,
		appdef.NullQName,
		cmdOnJoinedWorkspaceDeactivateExec,
	))

	adf.AddStruct(qNameProjectorApplyDeactivateWorkspace, appdef.DefKind_Object)

	// target app, target WSID
	cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameProjectorApplyDeactivateWorkspace,
			EventsFilter: []appdef.QName{qNameCmdDeactivateWorkspace},
			Func:         projectorApplyDeactivateWorkspace(federationURL, cfg.Name, tokensAPI, asp),
		}
	})
}

func cmdDeactivateWorkspaceExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
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
		return err
	}
	wsDescUpdater.PutInt32(sysshared.Field_Status, int32(sysshared.WorkspaceStatus_ToBeDeactivated))
	return nil
}

func cmdOnJoinedWorkspaceDeactivateExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	invitedToWSID := args.ArgumentObject.AsInt64(field_InvitedToWSID)
	svbCDocJoinedWorkspace, ok, err := invite.GetCDocJoinedWorkspaceForUpdate(args.State, args.Intents, invitedToWSID)
	if err != nil {
		return err
	}
	if ok {
		svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, false)
	}
	return nil
}

// app/pseudoProfileWSID, ownerApp
func cmdOnWorkspaceDeactivatedExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	ownerID := args.ArgumentObject.AsInt64(Field_OwnerID)
	kb, err := args.State.KeyBuilder(state.RecordsStorage, appdef.NullQName)
	if err != nil {
		// notest
		return err
	}
	kb.PutRecordID(state.Field_ID, istructs.RecordID(ownerID))
	ownerDoc, err := args.State.MustExist(kb)
	if err != nil {
		// notest
		return err
	}
	if !ownerDoc.AsBool(appdef.SystemField_IsActive) {
		return nil
	}
	ownerDocUpdater, err := args.Intents.UpdateValue(kb, ownerDoc)
	if err != nil {
		// notest
		return err
	}
	ownerDocUpdater.PutBool(appdef.SystemField_IsActive, false)
	return nil
}

// target app, target WSID
func projectorApplyDeactivateWorkspace(federationURL func() *url.URL, appQName istructs.AppQName, tokensAPI itokens.ITokens,
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
			// вызовем c.sys.OnJoinedWorkspaceDeactivated
			// по словам Максима: приложение всегда текщее
			// по словам Миши: в сабджектах не может быть логинов из разных приложений
			url := fmt.Sprintf(`api/%s/%d/c.sys.OnJoinedWorkspaceDeactivated`, appQName, profileWSID)

			body := fmt.Sprintf(`{"args":{"InvitedToWSID":%d}}`, event.Workspace())
			_, err = coreutils.FederationFunc(federationURL(), url, body, coreutils.WithAuthorizeBy(sysToken), coreutils.WithDiscardResponse())
			return err
		})
		if err != nil {
			// notestdebt
			return err
		}

		// c.sys.OnWorkspaceDeactivated(OnwerWSID, WSName) (appWS)
		wsName := wsDesc.AsString(sysshared.Field_WSName)
		body := fmt.Sprintf(`{"args":{"OwnerWSID":%d, "WSName":"%s"}}`, ownerID, wsName)
		cdocWorkspaceIDWSID := coreutils.GetAppWSID()
		if _, err := coreutils.FederationFunc(federationURL(), fmt.Sprintf("api/%s/%d/c.sys.OnWorkspaceDeactivated", ownerApp, ownerWSID), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken)); err != nil {
			return fmt.Errorf("c.sys.OnWorkspaceDeactivated failed: %w", err)
		}
		// c.sys.OnWorkspaceDeactivated(OwnerID) (profile)
		body := fmt.Sprintf(`{"args":{"OwnerID":%d}}`, ownerID)
		if _, err := coreutils.FederationFunc(federationURL(), fmt.Sprintf("api/%s/%d/c.sys.OnWorkspaceDeactivated", ownerApp, ownerWSID), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken)); err != nil {
			return fmt.Errorf("c.sys.OnWorkspaceDeactivated failed: %w", err)
		}

		// cdoc.sys.WorkspaceDescriptor.Status = Inactive
		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Status":%d}}]}`, wsDesc.AsRecordID(appdef.SystemField_ID), sysshared.WorkspaceStatus_Inactive)
		if _, err := coreutils.FederationFunc(federationURL(), fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken)); err != nil {
			return fmt.Errorf("c.sys.WorkspaceDescriptor.Status=Inactive by c.sys.CUD failed: %w", err)
		}
		logger.Info("workspace", wsDesc.AsString(sysshared.Field_WSName), "deactivated")
		return nil
	}
}
