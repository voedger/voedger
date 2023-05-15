/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package workspace

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/invite"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideDeactivateWorkspace(cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, tokensAPI itokens.ITokens, federationURL func() *url.URL) {

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
			AddField(field_WSName, appdef.DataKind_string, true).QName(),
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
			Func:         projectorApplyDeactivateWorkspace(federationURL, cfg.Name, tokensAPI),
		}
	})
}

func cmdDeactivateWorkspaceExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	kb, err := args.State.KeyBuilder(state.RecordsStorage, commandprocessor.QNameCDocWorkspaceDescriptor)
	if err != nil {
		// notest
		return err
	}
	kb.PutQName(state.Field_Singleton, commandprocessor.QNameCDocWorkspaceDescriptor)
	wsDesc, err := args.State.MustExist(kb)
	if err != nil {
		// notest
		return err
	}
	status := wsDesc.AsInt32(Field_Status)
	if status != int32(WorkspaceStatus_Active) {
		return coreutils.NewHTTPErrorf(http.StatusConflict, "Workspace Status is not Active")
	}

	wsDescUpdater, err := args.Intents.UpdateValue(kb, wsDesc)
	if err != nil {
		return err
	}
	wsDescUpdater.PutInt32(Field_Status, int32(WorkspaceStatus_ToBeDeactivated))
	return nil
}

func cmdOnJoinedWorkspaceDeactivateExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	wsName := args.ArgumentObject.AsString(field_WSName)
	kb, err := args.State.KeyBuilder(state.RecordsStorage, invite.QNameCDocJoinedWorkspace)
	if err != nil {
		return err
	}
	done := false
	return args.State.Read(kb, func(key istructs.IKey, value istructs.IStateValue) (err error) {
		if done || wsName != value.AsString(invite.Field_WSName) {
			return nil
		}
		done = true
		if value.AsBool(appdef.SystemField_IsActive) {
			return nil
		}
		cdocJoinedWorkspaceUpdater, err := args.Intents.UpdateValue(kb, value)
		if err != nil {
			// notest
			return err
		}
		cdocJoinedWorkspaceUpdater.PutBool(appdef.SystemField_IsActive, false)
		return nil
	})
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
func projectorApplyDeactivateWorkspace(federationURL func() *url.URL, appQName istructs.AppQName, tokensAPI itokens.ITokens) func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		kb, err := s.KeyBuilder(state.RecordsStorage, commandprocessor.QNameCDocWorkspaceDescriptor)
		if err != nil {
			// notest
			return err
		}
		kb.PutQName(state.Field_Singleton, commandprocessor.QNameCDocWorkspaceDescriptor)
		wsDesc, err := s.MustExist(kb)
		if err != nil {
			// notest
			return err
		}
		ownerApp := wsDesc.AsString(Field_OwnerApp)
		ownerWSID := wsDesc.AsInt64(Field_OwnerWSID)
		ownerDocID := wsDesc.AsInt64(Field_OwnerID)

		sysToken, err := payloads.GetSystemPrincipalToken(tokensAPI, appQName)
		if err != nil {
			// notest
			return err
		}

		// Foreach cdoc.sys.Subject
		subjectsKB, err := s.KeyBuilder(state.RecordsStorage, invite.QNameCDocSubject)
		if err != nil {
			// notest
			return err
		}
		s.Read(subjectsKB, func(key istructs.IKey, value istructs.IStateValue) (err error) {
			if istructs.SubjectKindType(value.AsInt32(invite.Field_SubjectKind)) != istructs.SubjectKind_User {
				return nil
			}
			profileWSID := istructs.WSID(value.AsInt64(invite.Field_ProfileWSID))
			// вызовем c.sys.OnJoinedWorkspaceDeactivated
			// по словам Максима: приложение всегда текщее
			// по словам Миши: в сабджектах не может быть логинов из разных приложений
			url := fmt.Sprintf(`api/%s/%d/c.sys.OnJoinedWorkspaceDeactivated`, appQName, profileWSID)

			wsName := wsDesc.AsString(authnz.Field_WSName)
			body := fmt.Sprintf(`{"args":{"WSName":"%s"}}`, wsName)
			_, err = coreutils.FederationFunc(federationURL(), url, body, coreutils.WithAuthorizeBy(sysToken), coreutils.WithDiscardResponse())
			return err
		})

		// c.sys.OnWorkspaceDeactivated(OwnerDocID)
		body := fmt.Sprintf(`{"args":{"OwnerID":%d}}`, ownerDocID)
		if _, err := coreutils.FederationFunc(federationURL(), fmt.Sprintf("api/%s/%d/c.sys.OnWorkspaceDeactivated", ownerApp, ownerWSID), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken)); err != nil {
			return fmt.Errorf("c.sys.OnWorkspaceDeactivated failed: %w", err)
		}

		// cdoc.sys.WorkspaceDescriptor.Status = Inactive
		wsDescUpdater, err := intents.UpdateValue(kb, wsDesc)
		if err != nil {
			// notest
			return err
		}
		wsDescUpdater.PutInt32(Field_Status, int32(WorkspaceStatus_Inactive))
		return nil
	}
}
