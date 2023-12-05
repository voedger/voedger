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
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc, tokensAPI itokens.ITokens,
	federation coreutils.IFederation, itokens itokens.ITokens, ep extensionpoints.IExtensionPoint, wsPostInitFunc WSPostInitFunc) {
	// c.sys.InitChildWorkspace
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		authnz.QNameCommandInitChildWorkspace,
		execCmdInitChildWorkspace,
	))

	// c.sys.CreateWorkspaceID
	// target app, (target cluster, base profile WSID)
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateWorkspaceID,
		execCmdCreateWorkspaceID(asp, cfg.Name),
	))

	// c.sys.CreateWorkspace
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateWorkspace,
		execCmdCreateWorkspace(timeFunc, asp, cfg.Name),
	))

	// q.sys.QueryChildWorkspaceByName
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryChildWorkspaceByName,
		qcwbnQryExec,
	))

	provideViewNextWSID(appDefBuilder)

	// deactivate workspace
	provideDeactivateWorkspace(cfg, tokensAPI, federation, asp)

	// projectors
	cfg.AddAsyncProjectors(
		provideAsyncProjectorFactoryInvokeCreateWorkspace(federation, cfg.Name, itokens),
		provideAsyncProjectorFactoryInvokeCreateWorkspaceID(federation, cfg.Name, itokens),
		provideAsyncProjectorInitializeWorkspace(federation, timeFunc, cfg.Name, ep, itokens, wsPostInitFunc),
	)
	cfg.AddSyncProjectors(
		provideSyncProjectorChildWorkspaceIdxFactory(),
		provideSyncProjectorWorkspaceIDIdx(),
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
func provideSyncProjectorWorkspaceIDIdx() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorViewWorkspaceIDIdx,
			Func: workspaceIDIdxProjector,
		}
	}
}
