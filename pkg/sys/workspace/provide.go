/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func Provide(sprb istructsmem.IStatelessPkgResourcesBuilder, timeFunc coreutils.TimeFunc, tokensAPI itokens.ITokens,
	federation federation.IFederation, itokens itokens.ITokens, ep extensionpoints.IExtensionPoint, wsPostInitFunc WSPostInitFunc) {
	// c.sys.InitChildWorkspace
	sprb.AddFunc(istructsmem.NewCommandFunction(
		authnz.QNameCommandInitChildWorkspace,
		execCmdInitChildWorkspace,
	))

	// c.sys.CreateWorkspaceID
	// target app, (target cluster, base profile WSID)
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateWorkspaceID,
		execCmdCreateWorkspaceID,
	))

	// c.sys.CreateWorkspace
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateWorkspace,
		execCmdCreateWorkspace(timeFunc),
	))

	// q.sys.QueryChildWorkspaceByName
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryChildWorkspaceByName,
		qcwbnQryExec,
	))

	provideViewNextWSID(cfg.AppDefBuilder())

	// deactivate workspace
	provideDeactivateWorkspace(cfg, tokensAPI, federation)

	// projectors
	cfg.AddAsyncProjectors(
		asyncProjectorInvokeCreateWorkspace(federation, itokens),
		asyncProjectorInvokeCreateWorkspaceID(federation, itokens),
		asyncProjectorInitializeWorkspace(federation, timeFunc, ep, itokens, wsPostInitFunc),
	)
	cfg.AddSyncProjectors(
		syncProjectorChildWorkspaceIdx(),
		syncProjectorWorkspaceIDIdx(),
	)
}

// proj.sys.ChildWorkspaceIdx
func syncProjectorChildWorkspaceIdx() istructs.Projector {
	return istructs.Projector{
		Name: QNameProjectorChildWorkspaceIdx,
		Func: childWorkspaceIdxProjector,
	}
}

// Projector<A, InitializeWorkspace>
func asyncProjectorInitializeWorkspace(federation federation.IFederation, nowFunc coreutils.TimeFunc, ep extensionpoints.IExtensionPoint,
	tokensAPI itokens.ITokens, wsPostInitFunc WSPostInitFunc) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPInitializeWorkspace,
		Func: initializeWorkspaceProjector(nowFunc, federation, ep, tokensAPI, wsPostInitFunc),
	}
}

// Projector<A, InvokeCreateWorkspaceID>
func asyncProjectorInvokeCreateWorkspaceID(federation federation.IFederation, tokensAPI itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPInvokeCreateWorkspaceID,
		Func: invokeCreateWorkspaceIDProjector(federation, tokensAPI),
	}
}

// Projector<A, InvokeCreateWorkspace>
func asyncProjectorInvokeCreateWorkspace(federation federation.IFederation, tokensAPI itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPInvokeCreateWorkspace,
		Func: invokeCreateWorkspaceProjector(federation, tokensAPI),
	}
}

// sp.sys.WorkspaceIDIdx
func syncProjectorWorkspaceIDIdx() istructs.Projector {
	return istructs.Projector{
		Name: QNameProjectorViewWorkspaceIDIdx,
		Func: workspaceIDIdxProjector,
	}
}
