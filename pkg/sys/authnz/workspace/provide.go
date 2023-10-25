/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc, tokensAPI itokens.ITokens,
	federation coreutils.IFederation, itokens itokens.ITokens, ep extensionpoints.IExtensionPoint, wsPostInitFunc WSPostInitFunc) {
	// c.sys.InitChildWorkspace
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		authnz.QNameCommandInitChildWorkspace,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "InitChildWorkspaceParams")).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKind, appdef.DataKind_QName, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, false).
			AddField(authnz.Field_WSClusterID, appdef.DataKind_int32, true).
			AddField(field_TemplateName, appdef.DataKind_string, false).
			AddField(Field_TemplateParams, appdef.DataKind_string, false).(appdef.IType).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdInitChildWorkspace,
	))

	// View<ChildWorkspaceIdx>
	// target app, user profile
	projectors.ProvideViewDef(appDefBuilder, QNameViewChildWorkspaceIdx, func(b appdef.IViewBuilder) {
		b.KeyBuilder().PartKeyBuilder().AddField(field_dummy, appdef.DataKind_int32)
		b.KeyBuilder().ClustColsBuilder().AddField(authnz.Field_WSName, appdef.DataKind_string)
		b.ValueBuilder().AddField(Field_ChildWorkspaceID, appdef.DataKind_int64, true)
	})

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
			AddField(Field_TemplateParams, appdef.DataKind_string, false).(appdef.IType).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdCreateWorkspaceID(asp, cfg.Name),
	))

	// View<WorkspaceIDIdx>
	projectors.ProvideViewDef(appDefBuilder, QNameViewWorkspaceIDIdx, func(b appdef.IViewBuilder) {
		b.KeyBuilder().PartKeyBuilder().AddField(Field_OwnerWSID, appdef.DataKind_int64)
		b.KeyBuilder().ClustColsBuilder().AddField(authnz.Field_WSName, appdef.DataKind_string)
		b.ValueBuilder().
			AddField(authnz.Field_WSID, appdef.DataKind_int64, true).
			AddRefField(field_IDOfCDocWorkspaceID, false) // TODO: not required for backward compatibility. Actually is required
	})

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
			AddField(Field_TemplateParams, appdef.DataKind_string, false).(appdef.IType).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdCreateWorkspace(timeFunc, asp, cfg.Name),
	))

	// q.sys.QueryChildWorkspaceByName
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryChildWorkspaceByName,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "QueryChildWorkspaceByNameParams")).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "QueryChildWorkspaceByNameResult")).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKind, appdef.DataKind_string, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, true).
			AddField(field_TemplateName, appdef.DataKind_string, true).
			AddField(Field_TemplateParams, appdef.DataKind_string, false).
			AddField(authnz.Field_WSID, appdef.DataKind_int64, false).
			AddField(authnz.Field_WSError, appdef.DataKind_string, false).
			AddField(appdef.SystemField_IsActive, appdef.DataKind_bool, true).(appdef.IType).QName(),
		qcwbnQryExec,
	))

	ProvideViewNextWSID(appDefBuilder)

	// deactivate workspace
	provideDeactivateWorkspace(cfg, appDefBuilder, tokensAPI, federation, asp)

	// projectors
	appDefBuilder.AddObject(qNameAPInvokeCreateWorkspace)
	appDefBuilder.AddObject(qNameAPInvokeCreateWorkspaceID)
	appDefBuilder.AddObject(qNameAPInitializeWorkspace)
	cfg.AddAsyncProjectors(
		provideAsyncProjectorFactoryInvokeCreateWorkspace(federation, cfg.Name, itokens),
		provideAsyncProjectorFactoryInvokeCreateWorkspaceID(federation, cfg.Name, itokens),
		provideAsyncProjectorInitializeWorkspace(federation, timeFunc, cfg.Name, ep, itokens, wsPostInitFunc),
	)
	cfg.AddSyncProjectors(
		provideSyncProjectorChildWorkspaceIdxFactory(),
		provideAsyncProjectorWorkspaceIDIdx(),
	)
}

// proj.sys.ChildWorkspaceIdx
func provideSyncProjectorChildWorkspaceIdxFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorChildWorkspaceIdx,
			Func: projectorChildWorkspaceIdx,
		}
	}
}

// Projector<A, InitializeWorkspace>
func provideAsyncProjectorInitializeWorkspace(federation coreutils.IFederation, nowFunc coreutils.TimeFunc, appQName istructs.AppQName, ep extensionpoints.IExtensionPoint,
	tokensAPI itokens.ITokens, wsPostInitFunc WSPostInitFunc) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPInitializeWorkspace,
			Func: initializeWorkspaceProjector(nowFunc, appQName, federation, ep, tokensAPI, wsPostInitFunc),
		}
	}
}

// Projector<A, InvokeCreateWorkspaceID>
func provideAsyncProjectorFactoryInvokeCreateWorkspaceID(federation coreutils.IFederation, appQName istructs.AppQName, tokensAPI itokens.ITokens) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPInvokeCreateWorkspaceID,
			Func: invokeCreateWorkspaceIDProjector(federation, appQName, tokensAPI),
		}
	}
}

// Projector<A, InvokeCreateWorkspace>
func provideAsyncProjectorFactoryInvokeCreateWorkspace(federation coreutils.IFederation, appQName istructs.AppQName, tokensAPI itokens.ITokens) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPInvokeCreateWorkspace,
			Func: invokeCreateWorkspaceProjector(federation, appQName, tokensAPI),
		}
	}
}

// sp.sys.WorkspaceIDIdx
func provideAsyncProjectorWorkspaceIDIdx() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorViewWorkspaceIDIdx,
			Func: workspaceIDIdxProjector,
		}
	}
}
