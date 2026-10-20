/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package btstrp

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/jsonu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	blobprocessor "github.com/voedger/voedger/pkg/processors/blobber"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
)

func Bootstrap(ctx context.Context, federationRetry federation.IFederationWithRetry, asp istructs.IAppStructsProvider, time timeu.ITime, appPartitions appparts.IAppPartitions,
	initialRecovery checkpoints.IInitialRecovery,
	clusterApp ClusterBuiltInApp, otherApps []appparts.BuiltInApp, sidecarApps []appparts.SidecarApp, itokens itokens.ITokens, storageProvider istorage.IAppStorageProvider,
	postWiredInterfacePtrs PostWireInterfacePtrs, blobHandler blobprocessor.IRequestHandler,
	requestSender bus.IRequestSender) (err error) {
	logCtx := logger.WithContextAttrs(ctx, map[string]any{
		logger.LogAttr_VApp:      sys.VApp_SysVoedger,
		logger.LogAttr_Extension: "sys._Bootstrap",
	})
	logger.InfoCtx(logCtx, "bootstrap", "started")

	// initialize cluster app workspace, use app ws amount 0
	if err := initClusterAppWS(asp, time); err != nil {
		return fmt.Errorf("initialize cluster app workspace: %w", err)
	}
	logger.InfoCtx(logCtx, "bootstrap", "cluster app workspace initialized")

	// Initialize AppStorageBlobber (* IAppStorage), AppStorageRouter (* IAppStorage)
	if *postWiredInterfacePtrs.BlobberAppStorage, err = storageProvider.AppStorage(istructs.AppQName_sys_blobber); err != nil {
		// notest
		return err
	}
	if *postWiredInterfacePtrs.RouterAppStorage, err = storageProvider.AppStorage(istructs.AppQName_sys_router); err != nil {
		// notest
		return err
	}

	*postWiredInterfacePtrs.RequestSender = requestSender

	*postWiredInterfacePtrs.BlobHandler = blobHandler

	// appparts: deploy single clusterApp partition
	appPartitions.DeployApp(istructs.AppQName_sys_cluster, nil, clusterApp.Def, clusterApp.AppDeploymentDescriptor.NumParts, clusterapp.ClusterAppNumEngines, clusterapp.ClusterAppNumAppWS)
	clusterPartitionIDs, err := initializeAppCheckpoints(logCtx, initialRecovery, appparts.BuiltInApp(clusterApp))
	if err != nil {
		return fmt.Errorf("initialize cluster app partition: %w", err)
	}
	if len(clusterPartitionIDs) == 0 {
		return errors.New("failed to initialize or deploy cluster app partition")
	}
	if err := deployAppPartitions(logCtx, "bootstrap.apppartdeploy.builtin", appPartitions, clusterApp.Name, clusterPartitionIDs); err != nil {
		return fmt.Errorf("deploy cluster app partition: %w", err)
	}

	sysToken, err := payloads.GetSystemPrincipalToken(itokens, istructs.AppQName_sys_cluster)
	if err != nil {
		return err
	}

	deployApp := func(appStage, partitionStage string, app appparts.BuiltInApp, extModuleURLs map[string]*url.URL) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		logger.InfoCtx(logCtx, appStage, app.Name)
		callDeployApp(federationRetry, sysToken, app)
		// Routing uses the full descriptor count even when some partitions cannot deploy.
		appPartitions.DeployApp(app.Name, extModuleURLs, app.Def, app.AppDeploymentDescriptor.NumParts, app.EnginePoolSize, app.NumAppWorkspaces)
		partitionIDs, err := initializeAppCheckpoints(logCtx, initialRecovery, app)
		if err != nil {
			return err
		}
		return deployAppPartitions(logCtx, partitionStage, appPartitions, app.Name, partitionIDs)
	}

	for _, app := range otherApps {
		if err := deployApp("bootstrap.appdeploy.builtin", "bootstrap.apppartdeploy.builtin", app, nil); err != nil {
			return err
		}
	}
	for _, sidecarApp := range sidecarApps {
		if err := deployApp("bootstrap.appdeploy.sidecar", "bootstrap.apppartdeploy.sidecar", sidecarApp.BuiltInApp, sidecarApp.ExtModuleURLs); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	logger.InfoCtx(logCtx, "bootstrap", "completed")
	return nil
}

func initializeAppCheckpoints(ctx context.Context, initialRecovery checkpoints.IInitialRecovery, app appparts.BuiltInApp) ([]istructs.PartitionID, error) {
	partitionIDsWithInitedCheckpoints, partitionIDsWithFailedCheckpoints, err := initialRecovery.InitializeCheckpoints(ctx, app.Name, app.AppDeploymentDescriptor.NumParts)
	for id, partitionErr := range partitionIDsWithFailedCheckpoints {
		errorCtx := logger.WithContextAttrs(ctx, map[string]any{
			logger.LogAttr_Extension: "sys._Recovery",
			"app":                    app.Name,
			"partid":                 id,
		})
		var workspaceErr *checkpoints.WorkspaceCheckpointError
		if errors.As(partitionErr, &workspaceErr) {
			errorCtx = logger.WithContextAttrs(errorCtx, map[string]any{logger.LogAttr_WSID: workspaceErr.WSID})
		}
		logger.ErrorCtx(errorCtx, "bootstrap.initial_recovery.error", partitionErr)
	}
	if err != nil {
		return nil, err
	}
	return partitionIDsWithInitedCheckpoints, nil
}

func deployAppPartitions(ctx context.Context, stage string, appPartitions appparts.IAppPartitions, appQName appdef.AppQName, partitionIDs []istructs.PartitionID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, id := range partitionIDs {
		logger.InfoCtx(ctx, stage, fmt.Sprintf("%s/%d", appQName, id))
	}
	if len(partitionIDs) > 0 {
		appPartitions.DeployAppPartitions(appQName, partitionIDs)
	}
	return nil
}

func callDeployApp(federationRetry federation.IFederationWithRetry, sysToken string, app appparts.BuiltInApp) {
	// Use Admin Endpoint to send requests
	body := jsonu.Jprintf(`{"args":{"AppQName":%q,"NumPartitions":%d,"NumAppWorkspaces":%d}}`, app.Name, app.AppDeploymentDescriptor.NumParts, app.NumAppWorkspaces)
	_, err := federationRetry.AdminFunc(fmt.Sprintf("api/%s/%d/c.cluster.DeployApp", istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID), body,
		httpu.WithDiscardResponse(),
		httpu.WithAuthorizeBy(sysToken),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to deploy app %s: %s", app.Name, err.Error()))
	}
}

func initClusterAppWS(asp istructs.IAppStructsProvider, time timeu.ITime) error {
	as, err := asp.BuiltIn(istructs.AppQName_sys_cluster)
	if err == nil {
		_, err = cluster.InitAppWS(as, clusterapp.ClusterAppWSIDPartitionID, clusterapp.ClusterAppWSID, istructs.FirstOffset, istructs.FirstOffset,
			istructs.UnixMilli(time.Now().UnixMilli()))
	}
	return err
}
