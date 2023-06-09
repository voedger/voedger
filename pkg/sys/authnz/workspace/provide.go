/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider, timeFunc func() time.Time, tokensAPI itokens.ITokens,
	federation coreutils.IFederation) {
	// c.sys.InitChildWorkspace
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		authnz.QNameCommandInitChildWorkspace,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "InitChildWorkspaceParams")).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKind, appdef.DataKind_QName, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, false).
			AddField(authnz.Field_WSClusterID, appdef.DataKind_int32, true).
			AddField(field_TemplateName, appdef.DataKind_string, false).
			AddField(Field_TemplateParams, appdef.DataKind_string, false).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdInitChildWorkspace,
	))

	// View<ChildWorkspaceIdx>
	// target app, user profile
	projectors.ProvideViewDef(appDefBuilder, QNameViewChildWorkspaceIdx, func(b appdef.IViewBuilder) {
		b.
			AddPartField(field_dummy, appdef.DataKind_int32).
			AddClustColumn(authnz.Field_WSName, appdef.DataKind_string).
			AddValueField(Field_ChildWorkspaceID, appdef.DataKind_int64, true)
	})

	// CDoc<ChildWorkspace>
	// many, target app, user profile
	appDefBuilder.AddCDoc(authnz.QNameCDocChildWorkspace).
		AddField(authnz.Field_WSName, appdef.DataKind_string, true).
		AddField(authnz.Field_WSKind, appdef.DataKind_QName, true).
		AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, false).
		AddField(field_TemplateName, appdef.DataKind_string, false).
		AddField(Field_TemplateParams, appdef.DataKind_string, false).
		AddField(authnz.Field_WSClusterID, appdef.DataKind_int32, true).
		AddField(authnz.Field_WSID, appdef.DataKind_int64, false).    // to be updated afterwards
		AddField(authnz.Field_WSError, appdef.DataKind_string, false) // to be updated afterwards

	// c.sys.CreateWorkspaceID
	// target app, (target cluster, base profile WSID)
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateWorkspaceID,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateWorkspaceIDParams")).
			AddField(Field_OwnerWSID, appdef.DataKind_int64, true).
			AddField(Field_OwnerQName, appdef.DataKind_QName, true).
			AddField(Field_OwnerID, appdef.DataKind_int64, true).
			AddField(Field_OwnerApp, appdef.DataKind_string, true).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKind, appdef.DataKind_QName, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, false).
			AddField(field_TemplateName, appdef.DataKind_string, false).
			AddField(Field_TemplateParams, appdef.DataKind_string, false).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdCreateWorkspaceID(asp, cfg.Name),
	))

	// View<WorkspaceIDIdx>
	projectors.ProvideViewDef(appDefBuilder, QNameViewWorkspaceIDIdx, func(b appdef.IViewBuilder) {
		b.
			AddPartField(Field_OwnerWSID, appdef.DataKind_int64).
			AddClustColumn(authnz.Field_WSName, appdef.DataKind_string).
			AddValueField(authnz.Field_WSID, appdef.DataKind_int64, true).
			AddValueField(field_IDOfCDocWorkspaceID, appdef.DataKind_RecordID, false) // TODO: not required for backward compatibility. Actually is required
	})

	// CDoc<WorkspaceID>
	// target app, (target cluster, base profile WSID)
	appDefBuilder.AddCDoc(QNameCDocWorkspaceID).
		AddField(Field_OwnerWSID, appdef.DataKind_int64, true).
		AddField(Field_OwnerQName, appdef.DataKind_QName, true).
		AddField(Field_OwnerID, appdef.DataKind_int64, true).
		AddField(Field_OwnerApp, appdef.DataKind_string, true).
		AddField(authnz.Field_WSName, appdef.DataKind_string, true).
		AddField(authnz.Field_WSKind, appdef.DataKind_QName, true).
		AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, false).
		AddField(field_TemplateName, appdef.DataKind_string, false).
		AddField(Field_TemplateParams, appdef.DataKind_string, false).
		AddField(authnz.Field_WSID, appdef.DataKind_int64, false)

	// c.sys.CreateWorkspace
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateWorkspace,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateWorkspaceParams")).
			AddField(Field_OwnerWSID, appdef.DataKind_int64, true).
			AddField(Field_OwnerQName, appdef.DataKind_QName, true).
			AddField(Field_OwnerID, appdef.DataKind_int64, true).
			AddField(Field_OwnerApp, appdef.DataKind_string, true).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKind, appdef.DataKind_QName, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, false).
			AddField(field_TemplateName, appdef.DataKind_string, false).
			AddField(Field_TemplateParams, appdef.DataKind_string, false).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdCreateWorkspace(timeFunc, asp, cfg.Name),
	))

	// singleton CDoc<sys.WorkspaceDescriptor>
	// target app, new WSID
	appDefBuilder.AddSingleton(authnz.QNameCDocWorkspaceDescriptor).
		AddField(Field_OwnerWSID, appdef.DataKind_int64, false). // owner* fields made non-required for app workspaces
		AddField(Field_OwnerQName, appdef.DataKind_QName, false).
		AddField(Field_OwnerID, appdef.DataKind_int64, false).
		AddField(Field_OwnerApp, appdef.DataKind_string, false). // QName -> each target app must know the owner QName -> string
		AddField(authnz.Field_WSName, appdef.DataKind_string, true).
		AddField(authnz.Field_WSKind, appdef.DataKind_QName, true).
		AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, false).
		AddField(field_TemplateName, appdef.DataKind_string, false).
		AddField(Field_TemplateParams, appdef.DataKind_string, false).
		AddField(authnz.Field_WSID, appdef.DataKind_int64, false).
		AddField(Field_CreateError, appdef.DataKind_string, false).
		AddField(authnz.Field_СreatedAtMs, appdef.DataKind_int64, true).
		AddField(Field_InitStartedAtMs, appdef.DataKind_int64, false).
		AddField(Field_InitError, appdef.DataKind_string, false).
		AddField(Field_InitCompletedAtMs, appdef.DataKind_int64, false).
		AddField(Field_Status, appdef.DataKind_int32, false)

	// q.sys.QueryChildWorkspaceByName
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryChildWorkspaceByName,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "QueryChildWorkspaceByNameParams")).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).(appdef.IDef).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "QueryChildWorkspaceByNameResult")).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKind, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, true).
			AddField(field_TemplateName, appdef.DataKind_string, true).
			AddField(Field_TemplateParams, appdef.DataKind_string, false).
			AddField(authnz.Field_WSID, appdef.DataKind_int64, false).
			AddField(authnz.Field_WSError, appdef.DataKind_string, false).
			AddField(appdef.SystemField_IsActive, appdef.DataKind_bool, true).(appdef.IDef).QName(),
		qcwbnQryExec,
	))

	//register asynchronous projector QNames
	appDefBuilder.AddObject(qNameAPInitializeWorkspace)
	appDefBuilder.AddObject(qNameAPInvokeCreateWorkspaceID)
	appDefBuilder.AddObject(qNameAPInvokeCreateWorkspace)

	ProvideViewNextWSID(appDefBuilder)

	// deactivate workspace
	provideDeactivateWorkspace(cfg, appDefBuilder, tokensAPI, federation, asp)
}

// proj.sys.ChildWorkspaceIdx
func ProvideSyncProjectorChildWorkspaceIdxFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameViewChildWorkspaceIdx,
			Func: ChildWorkspaceIdxProjector,
		}
	}
}

// Projector<A, InitializeWorkspace>
func ProvideAsyncProjectorInitializeWorkspace(federation coreutils.IFederation, nowFunc func() time.Time, appQName istructs.AppQName, ep extensionpoints.IExtensionPoint,
	tokensAPI itokens.ITokens, wsPostInitFunc WSPostInitFunc) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPInitializeWorkspace,
			Func: initializeWorkspaceProjector(nowFunc, appQName, federation, ep, tokensAPI, wsPostInitFunc),
		}
	}
}

// Projector<A, InvokeCreateWorkspaceID>
func ProvideAsyncProjectorFactoryInvokeCreateWorkspaceID(federation coreutils.IFederation, appQName istructs.AppQName, tokensAPI itokens.ITokens) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPInvokeCreateWorkspaceID,
			Func: invokeCreateWorkspaceIDProjector(federation, appQName, tokensAPI),
		}
	}
}

// Projector<A, InvokeCreateWorkspace>
func ProvideAsyncProjectorFactoryInvokeCreateWorkspace(federation coreutils.IFederation, appQName istructs.AppQName, tokensAPI itokens.ITokens) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPInvokeCreateWorkspace,
			Func: invokeCreateWorkspaceProjector(federation, appQName, tokensAPI),
		}
	}
}

// sp.sys.WorkspaceIDIdx
func ProvideAsyncProjectorWorkspaceIDIdx() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameViewWorkspaceIDIdx,
			Func: workspaceIDIdxProjector,
		}
	}
}
