/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/btstrp"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/checkpoints/testutils"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iblobstoragestg"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/parser"
	blobprocessor "github.com/voedger/voedger/pkg/processors/blobber"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	"github.com/voedger/voedger/pkg/sys/uniques"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
	"github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
	dbcertcache "github.com/voedger/voedger/pkg/vvm/db_cert_cache"
	vvmstorage "github.com/voedger/voedger/pkg/vvm/storage"
)

func TestBoostrap_BasicUsage(t *testing.T) {
	require := require.New(t)

	// launch the VVM with an app with a certain NumParts and NumAppWorkspaces
	numParts := istructs.NumAppPartitions(42)
	numAppWS := istructs.NumAppWorkspaces(43)
	cfg := getTestVITCfg(numParts, numAppWS)
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	clusterApp, otherApps := bootstrapTestApps(vit)

	t.Run("basic usage", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		postWiredInterfacePtrs := newPostWiredInterfacePtrs()
		testBlobRequestHandler := blobprocessor.NewIRequestHandler(nil, 0, nil)
		testRequestSender := bus.NewIRequestSender(testingu.MockTime, nil)
		err := btstrp.Bootstrap(t.Context(), vit.IFederation.WithRetry(), vit.IAppStructsProvider, vit.Time, appParts, newMockInitialRecovery(vit, nil), clusterApp, otherApps,
			nil, vit.ITokens, vit.IAppStorageProvider, postWiredInterfacePtrs, testBlobRequestHandler, testRequestSender)
		require.NoError(err)
		require.NotNil(*postWiredInterfacePtrs.BlobberAppStorage)
		require.NotNil(*postWiredInterfacePtrs.RouterAppStorage)
		require.NotNil(*postWiredInterfacePtrs.RequestSender)
		require.NotNil(*postWiredInterfacePtrs.BlobHandler)

	})

	t.Run("panic on NumPartitions change", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		otherApps[0].AppDeploymentDescriptor.NumParts++
		defer func() {
			otherApps[0].AppDeploymentDescriptor.NumParts--
		}()
		postWiredInterfacePtrs := newPostWiredInterfacePtrs()
		testBlobRequestHandler := blobprocessor.NewIRequestHandler(nil, 0, nil)
		testRequestSender := bus.NewIRequestSender(testingu.MockTime, nil)
		//nolint errcheck
		require.PanicsWithValue(fmt.Sprintf("failed to deploy app %[1]s: status 409, expected [200 201]: num partitions changed: app %[1]s declaring NumPartitions=%d but was previously deployed with NumPartitions=%d",
			otherApps[0].Name, otherApps[0].AppDeploymentDescriptor.NumParts, otherApps[0].AppDeploymentDescriptor.NumParts-1), func() {
			btstrp.Bootstrap(t.Context(), vit.IFederation.WithRetry(), vit.IAppStructsProvider, vit.Time, appParts, newMockInitialRecovery(vit, nil), clusterApp, otherApps,
				nil, vit.ITokens, vit.IAppStorageProvider, postWiredInterfacePtrs, testBlobRequestHandler, testRequestSender)
		})
	})

	t.Run("panic on NumAppPartitions change", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		otherApps[0].AppDeploymentDescriptor.NumAppWorkspaces++
		defer func() {
			otherApps[0].AppDeploymentDescriptor.NumAppWorkspaces--
		}()

		//nolint errcheck
		require.PanicsWithValue(fmt.Sprintf("failed to deploy app %[1]s: status 409, expected [200 201]: num application workspaces changed: app %[1]s declaring NumAppWorkspaces=%d but was previously deployed with NumAppWorkspaces=%d",
			otherApps[0].Name, otherApps[0].AppDeploymentDescriptor.NumAppWorkspaces, otherApps[0].AppDeploymentDescriptor.NumAppWorkspaces-1), func() {
			postWiredInterfacePtrs := newPostWiredInterfacePtrs()
			testBlobRequestHandler := blobprocessor.NewIRequestHandler(nil, 0, nil)
			testRequestSender := bus.NewIRequestSender(testingu.MockTime, nil)
			btstrp.Bootstrap(t.Context(), vit.IFederation.WithRetry(), vit.IAppStructsProvider, vit.Time, appParts, newMockInitialRecovery(vit, nil), clusterApp, otherApps,
				nil, vit.ITokens, vit.IAppStorageProvider, postWiredInterfacePtrs, testBlobRequestHandler, testRequestSender)
		})
	})
}

func TestBootstrapInitialRecoveryOrchestration(t *testing.T) {
	numParts := istructs.NumAppPartitions(3)
	numAppWS := istructs.NumAppWorkspaces(1)
	cfg := getTestVITCfg(numParts, numAppWS)
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()
	clusterApp, otherApps := bootstrapTestApps(vit)
	require.NotEmpty(t, otherApps)
	var sidecarApps []appparts.SidecarApp
	for index, app := range otherApps {
		if app.Name == istructs.AppQName_test1_app1 {
			sidecarApps = []appparts.SidecarApp{{BuiltInApp: app}}
			otherApps = append(otherApps[:index:index], otherApps[index+1:]...)
			break
		}
	}
	require.Len(t, sidecarApps, 1)
	targetApp := sidecarApps[0].Name

	t.Run("initializes every registered app before its partitions", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		bootstrapRecorder := &bootstrapOperationsRecorder{}
		appPartsRecorder := &appPartsRecorder{IAppPartitions: appParts, bootstrapRecorder: bootstrapRecorder}
		iInitialRecovery := newMockInitialRecovery(vit, bootstrapRecorder)

		err := runBootstrap(t.Context(), vit, appPartsRecorder, iInitialRecovery, clusterApp, otherApps, sidecarApps)

		require.NoError(t, err)
		apps := []appparts.BuiltInApp{appparts.BuiltInApp(clusterApp)}
		apps = append(apps, otherApps...)
		apps = append(apps, sidecarApps[0].BuiltInApp)

		// check operations order: DeployApp, InitializeCheckpoints, DeployAppPartitions
		for _, app := range apps {
			order_DeployApp := bootstrapRecorder.requireOrder(operationDeployApp, app.Name)
			order_InitializeCheckpoints := bootstrapRecorder.requireOrder(operationInitializeCheckpoints, app.Name)
			order_DeployAppPartitions := bootstrapRecorder.requireOrder(operationDeployAppPartitions, app.Name)
			require.Less(t, order_DeployApp, order_InitializeCheckpoints, app.Name.String())
			require.Less(t, order_InitializeCheckpoints, order_DeployAppPartitions, app.Name.String())
			require.Len(t, bootstrapRecorder.deployedPartitionIDs(app.Name), int(app.NumParts))
			require.ElementsMatch(t, iInitialRecovery.deployable[app.Name], bootstrapRecorder.deployedPartitionIDs(app.Name))
		}
	})

	for _, test := range []struct {
		name       string
		failed     []istructs.PartitionID
		deployable []istructs.PartitionID
	}{
		{"one partition", []istructs.PartitionID{1}, []istructs.PartitionID{0, 2}},
		{"all partitions", []istructs.PartitionID{0, 1, 2}, nil},
	} {
		t.Run(test.name+" failures exclude partitions without stopping readiness", func(t *testing.T) {
			appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
			defer cleanup()
			bootstrapRecorder := &bootstrapOperationsRecorder{}
			appPartsRecorder := &appPartsRecorder{IAppPartitions: appParts, bootstrapRecorder: bootstrapRecorder}
			iInitialRecovery := newMockInitialRecovery(vit, bootstrapRecorder)
			iInitialRecovery.doFailPartitionCheckpointRead = map[appdef.AppQName]map[istructs.PartitionID]error{
				targetApp: {},
			}
			for _, id := range test.failed {
				iInitialRecovery.doFailPartitionCheckpointRead[targetApp][id] = errors.New("checkpoint read failed")
			}
			logCap := logger.StartCapture(t, logger.LogLevelVerbose)
			appPartsRecorder.beforeDeploy = func(app appdef.AppQName) {
				if app != targetApp {
					return
				}
				for _, id := range test.failed {
					logCap.HasLine("stage=bootstrap.initial_recovery.error", "app="+targetApp.String(),
						fmt.Sprintf("partid=%d", id), "checkpoint read failed")
				}
			}

			err := runBootstrap(t.Context(), vit, appPartsRecorder, iInitialRecovery, clusterApp, otherApps, sidecarApps)

			require.NoError(t, err)
			require.ElementsMatch(t, test.deployable, bootstrapRecorder.deployedPartitionIDs(targetApp))
			for _, id := range test.failed {
				logCap.HasLine("stage=bootstrap.initial_recovery.error", "level=ERROR", "app="+targetApp.String(),
					fmt.Sprintf("partid=%d", id), "get partition checkpoint", "checkpoint read failed")
				require.Equal(t, 1, countBootstrapLogLines(logCap.String(), "stage=bootstrap.initial_recovery.error",
					"app="+targetApp.String(), fmt.Sprintf("partid=%d", id)), "bootstrap and initializer must not log twice")

				// can not borrow because an errored partition is not deployed
				partition, err := appParts.Borrow(targetApp, id, appparts.ProcessorKind_Command)
				require.Nil(t, partition)
				require.ErrorIs(t, err, appparts.ErrNotFound)
			}
		})
	}

	t.Run("recovery: scn: Cluster partition failure stops bootstrap", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		bootstrapRecorder := &bootstrapOperationsRecorder{}
		appPartsRecorder := &appPartsRecorder{IAppPartitions: appParts, bootstrapRecorder: bootstrapRecorder}
		iInitialRecovery := newMockInitialRecovery(vit, bootstrapRecorder)
		// Given initial recovery fails for the cluster application's sole partition
		iInitialRecovery.doFailPartitionCheckpointRead = map[appdef.AppQName]map[istructs.PartitionID]error{
			istructs.AppQName_sys_cluster: {0: errors.New("cluster app checkpoint unavailable")},
		}

		federationRecorder := &federationRecorder{IFederationWithRetry: vit.IFederation.WithRetry()}
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// When VAOperator starts the VVM
		err := runBootstrap(t.Context(), vit, appPartsRecorder, iInitialRecovery, clusterApp, otherApps, sidecarApps, federationRecorder)

		// Then bootstrap returns a cluster partition initialization or deployment error
		require.ErrorContains(t, err, "failed to initialize or deploy cluster app partition")
		federationRecorder.requireNoDeploymentCommands(t)
		logCap.HasLine("stage=bootstrap.initial_recovery.error", "app=sys/cluster", "partid=0", "cluster app checkpoint unavailable")
		// And the server does not become ready
		logCap.NotContains("stage=bootstrap.appdeploy", "msg=completed")
		// And no other application is registered or provisioned
		for _, app := range append(otherApps, sidecarApps[0].BuiltInApp) {
			_, err := appParts.AppPartsCount(app.Name)
			require.ErrorIs(t, err, appparts.ErrNotFound, "later apps must not be registered")
		}

		// The failed cluster partition remains undeployed.
		partition, err := appParts.Borrow(istructs.AppQName_sys_cluster, 0, appparts.ProcessorKind_Command)
		require.Nil(t, partition)
		require.ErrorIs(t, err, appparts.ErrNotFound)
	})

	t.Run("failed to get IAppStructs -> no partition deploy", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		bootstrapRecorder := &bootstrapOperationsRecorder{}
		appPartsRecorder := &appPartsRecorder{IAppPartitions: appParts, bootstrapRecorder: bootstrapRecorder}
		getIAppStructsErr := errors.New("IAppStructs unavailable")
		initializer := newMockInitialRecovery(vit, bootstrapRecorder)
		initializer.doFailAppStructsGet = map[appdef.AppQName]error{
			otherApps[0].Name: getIAppStructsErr,
		}

		err := runBootstrap(t.Context(), vit, appPartsRecorder, initializer, clusterApp, otherApps, sidecarApps)

		require.ErrorIs(t, err, getIAppStructsErr)
		for id := istructs.PartitionID(0); id < istructs.PartitionID(otherApps[0].NumParts); id++ {
			partition, err := appParts.Borrow(otherApps[0].Name, id, appparts.ProcessorKind_Command)
			require.Nil(t, partition)
			require.ErrorIs(t, err, appparts.ErrNotFound)
		}
	})

	t.Run("bootstrap cancellation stops partition deployment", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		bootstrapRecorder := &bootstrapOperationsRecorder{}
		recordingParts := &appPartsRecorder{IAppPartitions: appParts, bootstrapRecorder: bootstrapRecorder}
		initializer := newMockInitialRecovery(vit, bootstrapRecorder)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := runBootstrap(ctx, vit, recordingParts, initializer, clusterApp, otherApps, sidecarApps)

		require.ErrorIs(t, err, context.Canceled)
		partition, err := appParts.Borrow(istructs.AppQName_sys_cluster, 0, appparts.ProcessorKind_Command)
		require.Nil(t, partition)
		require.ErrorIs(t, err, appparts.ErrNotFound)
	})

	t.Run("cancellation after a completed partition prevents partial deployment", func(t *testing.T) {
		appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
		defer cleanup()
		bootstrapRecorder := &bootstrapOperationsRecorder{}
		parts := &appPartsRecorder{IAppPartitions: appParts, bootstrapRecorder: bootstrapRecorder}
		iInitialRecovery := newMockInitialRecovery(vit, bootstrapRecorder)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		iInitialRecovery.checkpointReadHook = func(app appdef.AppQName, id istructs.PartitionID) {
			if app == targetApp && id == 0 {
				cancel()
			}
		}
		err := runBootstrap(ctx, vit, parts, iInitialRecovery, clusterApp, otherApps, sidecarApps)
		require.ErrorIs(t, err, context.Canceled)
		for id := istructs.PartitionID(0); id < istructs.PartitionID(numParts); id++ {
			partition, err := appParts.Borrow(targetApp, id, appparts.ProcessorKind_Command)
			require.Nil(t, partition)
			require.ErrorIs(t, err, appparts.ErrNotFound)
		}
	})
}

func TestBootstrapCheckpointFailuresPreserveDeclaredPartitionCount(t *testing.T) {
	for _, test := range []struct {
		name       string
		failed     []istructs.PartitionID
		deployable []istructs.PartitionID
	}{
		{"one failed partition", []istructs.PartitionID{1}, []istructs.PartitionID{0, 2}},
		{"all failed partitions", []istructs.PartitionID{0, 1, 2}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := getTestVITCfg(3, 1)
			vit := it.NewVIT(t, &cfg)
			defer vit.TearDown()
			clusterApp, apps := bootstrapTestApps(vit)
			var app appparts.BuiltInApp
			for _, candidate := range apps {
				if candidate.Name == istructs.AppQName_test1_app1 {
					app = candidate
				}
			}
			require.Equal(t, istructs.NumAppPartitions(3), app.NumParts)
			// Add application structures without a cluster deployment descriptor, so
			// the first command exercises actual descriptor creation and provisioning.
			app.Name = istructs.AppQName_test1_app2
			_, err := vit.IAppStructsProvider.New(app.Name, app.Def, istructs.ClusterApps[app.Name], app.NumAppWorkspaces)
			require.NoError(t, err)
			_, exists := findAppDeploymentDescriptor(t, vit.IAppStructsProvider, app.Name)
			require.False(t, exists)
			var storedDescriptor map[string]interface{}

			for _, phase := range []string{"first deployment", "repeated startup"} {
				t.Run(phase, func(t *testing.T) {
					appParts, cleanup := appparts.NewTestAppParts(vit.IAppStructsProvider)
					defer cleanup()
					recorder := &bootstrapOperationsRecorder{}
					parts := &appPartsRecorder{IAppPartitions: appParts, bootstrapRecorder: recorder}
					initializer := newMockInitialRecovery(vit, recorder)
					initializer.doFailPartitionCheckpointRead = map[appdef.AppQName]map[istructs.PartitionID]error{app.Name: {}}
					for _, id := range test.failed {
						initializer.doFailPartitionCheckpointRead[app.Name][id] = errors.New("checkpoint unavailable")
					}
					federationRecorder := &federationRecorder{IFederationWithRetry: vit.IFederation.WithRetry()}

					require.NoError(t, runBootstrap(t.Context(), vit, parts, initializer, clusterApp, []appparts.BuiltInApp{app}, nil, federationRecorder))

					federationRecorder.requireDeploymentPartitionCount(t, app.Name, 3)
					recorder.requireRegisteredPartitionCount(t, app.Name, 3)
					count, err := appParts.AppPartsCount(app.Name)
					require.NoError(t, err)
					require.Equal(t, istructs.NumAppPartitions(3), count)
					descriptor, exists := findAppDeploymentDescriptor(t, vit.IAppStructsProvider, app.Name)
					require.True(t, exists)
					require.EqualValues(t, 3, descriptor.AsInt32(cluster.Field_NumPartitions))
					fields := deploymentDescriptorFields(t, vit.IAppStructsProvider, app.Name)
					if storedDescriptor == nil {
						storedDescriptor = fields
					} else {
						require.Equal(t, storedDescriptor, fields, "compatible startup must retain the complete descriptor")
					}
					require.ElementsMatch(t, test.deployable, recorder.deployedPartitionIDs(app.Name))
					if len(test.deployable) == 0 {
						recorder.requireNoPartitionDeployment(t, app.Name)
					}
				})
			}
		})
	}
}

type deploymentCommand struct {
	AppQName      string
	NumPartitions istructs.NumAppPartitions
}

type federationRecorder struct {
	federation.IFederationWithRetry
	deploymentCommands []deploymentCommand
}

type vvmStorageKeyCaptorOnGet struct {
	vvmstorage.ISysVvmStorage
	key []byte
}

func (s *vvmStorageKeyCaptorOnGet) Get(key, _ []byte, _ *[]byte) (bool, error) {
	s.key = bytes.Clone(key)
	return false, nil
}

func (f *federationRecorder) AdminFunc(relativeURL, body string, opts ...httpu.ReqOptFunc) (*federation.FuncResponse, error) {
	var request struct{ Args deploymentCommand }
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		return nil, fmt.Errorf("decode deployment command: %w", err)
	}
	f.deploymentCommands = append(f.deploymentCommands, request.Args)
	return f.IFederationWithRetry.AdminFunc(relativeURL, body, opts...)
}

func (f *federationRecorder) requireNoDeploymentCommands(t *testing.T) {
	t.Helper()
	require.Empty(t, f.deploymentCommands, "AdminFunc must not be called")
}

func (f *federationRecorder) requireDeploymentPartitionCount(t *testing.T, app appdef.AppQName, count istructs.NumAppPartitions) {
	t.Helper()
	var commands []deploymentCommand
	for _, command := range f.deploymentCommands {
		if command.AppQName == app.String() {
			commands = append(commands, command)
		}
	}
	require.Equal(t, []deploymentCommand{{AppQName: app.String(), NumPartitions: count}}, commands)
}

func findAppDeploymentDescriptor(t *testing.T, asp istructs.IAppStructsProvider, app appdef.AppQName) (istructs.IRecord, bool) {
	t.Helper()
	structs, err := asp.BuiltIn(istructs.AppQName_sys_cluster)
	require.NoError(t, err)
	id, err := uniques.GetRecordIDByUniqueCombination(clusterapp.ClusterAppWSID, appdef.NewQName(cluster.ClusterPackage, "App"), structs,
		map[string]interface{}{cluster.Field_AppQName: app.String()})
	require.NoError(t, err)
	if id == istructs.NullRecordID {
		return nil, false
	}
	record, err := structs.Records().Get(clusterapp.ClusterAppWSID, true, id)
	require.NoError(t, err)
	return record, true
}

func deploymentDescriptorFields(t *testing.T, asp istructs.IAppStructsProvider, app appdef.AppQName) map[string]interface{} {
	t.Helper()
	record, exists := findAppDeploymentDescriptor(t, asp, app)
	require.True(t, exists)
	structs, err := asp.BuiltIn(istructs.AppQName_sys_cluster)
	require.NoError(t, err)
	return coreutils.FieldsToMap(record, structs.AppDef())
}

func countBootstrapLogLines(logs string, fragments ...string) int {
	count := 0
	for line := range strings.SplitSeq(logs, "\n") {
		matches := true
		for _, fragment := range fragments {
			matches = matches && strings.Contains(line, fragment)
		}
		if matches {
			count++
		}
	}
	return count
}

func bootstrapTestApps(vit *it.VIT) (btstrp.ClusterBuiltInApp, []appparts.BuiltInApp) {
	vit.T.Helper()
	builtinApps := make([]appparts.BuiltInApp, 0, len(vit.BuiltInAppsPackages))
	for _, app := range vit.BuiltInAppsPackages {
		builtinApps = append(builtinApps, app.BuiltInApp)
	}
	clusterApp, otherApps, err := btstrp.SplitBuiltInApps(builtinApps, nil)
	require.NoError(vit.T, err)
	return clusterApp, otherApps
}

func runBootstrap(ctx context.Context, vit *it.VIT, appParts appparts.IAppPartitions,
	iInitialRecovery checkpoints.IInitialRecovery, clusterApp btstrp.ClusterBuiltInApp,
	otherApps []appparts.BuiltInApp, sidecarApps []appparts.SidecarApp,
	federations ...federation.IFederationWithRetry) error {
	postWiredInterfacePtrs := newPostWiredInterfacePtrs()
	testBlobRequestHandler := blobprocessor.NewIRequestHandler(nil, 0, nil)
	testRequestSender := bus.NewIRequestSender(testingu.MockTime, nil)
	federationWithRetry := vit.IFederation.WithRetry()
	if len(federations) > 0 {
		federationWithRetry = federations[0]
	}
	return btstrp.Bootstrap(ctx, federationWithRetry, vit.IAppStructsProvider, vit.Time, appParts, iInitialRecovery,
		clusterApp, otherApps, sidecarApps, vit.ITokens, vit.IAppStorageProvider, postWiredInterfacePtrs,
		testBlobRequestHandler, testRequestSender)
}

type bootstrapOperation int

const (
	operationDeployApp bootstrapOperation = iota
	operationInitializeCheckpoints
	operationDeployAppPartitions
)

type bootstrapAppOperation struct {
	operation      bootstrapOperation
	appQName       appdef.AppQName
	partitionIDs   []istructs.PartitionID    // for operationDeployAppPartitions only
	partitionCount istructs.NumAppPartitions // for operationDeployApp only
}

// need to track the correct order of methods call:
// DeployApp() -> InitializeCheckpoints() -> DeployAppPartitions()
type bootstrapOperationsRecorder struct {
	mu         sync.Mutex
	operations []bootstrapAppOperation
}

func (l *bootstrapOperationsRecorder) append(operation bootstrapOperation, appQName appdef.AppQName, ids ...istructs.PartitionID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.operations = append(l.operations, bootstrapAppOperation{operation: operation, appQName: appQName,
		partitionIDs: append([]istructs.PartitionID(nil), ids...)})
}

func (l *bootstrapOperationsRecorder) recordAppRegistration(app appdef.AppQName, count istructs.NumAppPartitions) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.operations = append(l.operations, bootstrapAppOperation{operation: operationDeployApp, appQName: app, partitionCount: count})
}

func (l *bootstrapOperationsRecorder) requireRegisteredPartitionCount(t *testing.T, app appdef.AppQName, count istructs.NumAppPartitions) {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, operation := range l.operations {
		if operation.operation == operationDeployApp && operation.appQName == app {
			require.Equal(t, count, operation.partitionCount)
			return
		}
	}
	t.Fatalf("application %s was not registered", app)
}

func (l *bootstrapOperationsRecorder) requireNoPartitionDeployment(t *testing.T, app appdef.AppQName) {
	t.Helper()
	require.Equal(t, -1, l.index(operationDeployAppPartitions, app), "DeployAppPartitions must not be called for %s", app)
}

func (l *bootstrapOperationsRecorder) deployedPartitionIDs(app appdef.AppQName) []istructs.PartitionID {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, operation := range l.operations {
		if operation.operation == operationDeployAppPartitions && operation.appQName == app {
			return append([]istructs.PartitionID(nil), operation.partitionIDs...)
		}
	}
	return nil
}

func (l *bootstrapOperationsRecorder) requireOrder(operation bootstrapOperation, appQName appdef.AppQName) int {
	index := l.index(operation, appQName)
	if index < 0 {
		panic(fmt.Sprintf("bootstrap operation %d was not made for app %s", operation, appQName))
	}
	return index
}

func (l *bootstrapOperationsRecorder) index(operation bootstrapOperation, appQName appdef.AppQName) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	for index, actual := range l.operations {
		if actual.operation == operation && actual.appQName == appQName {
			return index
		}
	}
	return -1
}

type appPartsRecorder struct {
	appparts.IAppPartitions
	bootstrapRecorder *bootstrapOperationsRecorder
	beforeDeploy      func(appdef.AppQName)
}

func (p *appPartsRecorder) DeployApp(name appdef.AppQName, extModuleURLs map[string]*url.URL,
	def appdef.IAppDef, partsCount istructs.NumAppPartitions, numEngines [appparts.ProcessorKind_Count]uint,
	numAppWorkspaces istructs.NumAppWorkspaces) {
	p.bootstrapRecorder.recordAppRegistration(name, partsCount)
	p.IAppPartitions.DeployApp(name, extModuleURLs, def, partsCount, numEngines, numAppWorkspaces)
}

func (p *appPartsRecorder) DeployAppPartitions(name appdef.AppQName, partitionIDs []istructs.PartitionID) {
	if p.beforeDeploy != nil {
		p.beforeDeploy(name)
	}
	p.bootstrapRecorder.append(operationDeployAppPartitions, name, partitionIDs...)
	p.IAppPartitions.DeployAppPartitions(name, partitionIDs)
}

type mockInitialRecovery struct {
	checkpoints.IInitialRecovery
	mu                            sync.Mutex
	bootstrapRecorder             *bootstrapOperationsRecorder
	doFailAppStructsGet           map[appdef.AppQName]error
	doFailPartitionCheckpointRead map[appdef.AppQName]map[istructs.PartitionID]error
	appQNamesByID                 map[istructs.ClusterAppID]appdef.AppQName
	deployable                    map[appdef.AppQName][]istructs.PartitionID
	checkpointReadHook            func(appdef.AppQName, istructs.PartitionID)
}

var _ checkpoints.IInitialRecovery = (*mockInitialRecovery)(nil)

func newMockInitialRecovery(vit *it.VIT, bootstrapRecorder *bootstrapOperationsRecorder) *mockInitialRecovery {
	sysVVMStorage, err := vit.IAppStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(vit.T, err)
	recovery := &mockInitialRecovery{
		bootstrapRecorder: bootstrapRecorder,
		appQNamesByID:     map[istructs.ClusterAppID]appdef.AppQName{},
		deployable:        map[appdef.AppQName][]istructs.PartitionID{},
	}
	appStructsProvider := &mockInitialRecoveryAppStructsProvider{
		IAppStructsProvider: vit.IAppStructsProvider,
		recovery:            recovery,
	}
	checkpointStorage := testutils.WrapStorage(vvmstorage.NewRecoveryCheckpointStorage(sysVVMStorage))
	checkpointStorage.SetBefore(testutils.GetPartition, func(op testutils.Operation) error {
		recovery.mu.Lock()
		defer recovery.mu.Unlock()
		appQName := recovery.appQNamesByID[op.AppID]
		return recovery.doFailPartitionCheckpointRead[appQName][op.PartitionID]
	})
	checkpointStorage.SetAfter(testutils.GetPartition, func(op testutils.Operation) {
		recovery.mu.Lock()
		appQName := recovery.appQNamesByID[op.AppID]
		hook := recovery.checkpointReadHook
		recovery.mu.Unlock()
		if hook != nil {
			hook(appQName, op.PartitionID)
		}
	})
	recovery.IInitialRecovery = checkpoints.NewInitialRecovery(appStructsProvider, checkpointStorage)
	return recovery
}

func (r *mockInitialRecovery) InitializeCheckpoints(ctx context.Context, appQName appdef.AppQName,
	numPartitions istructs.NumAppPartitions) ([]istructs.PartitionID, map[istructs.PartitionID]error, error) {
	deployable, failed, err := r.IInitialRecovery.InitializeCheckpoints(ctx, appQName, numPartitions)
	r.mu.Lock()
	r.deployable[appQName] = append([]istructs.PartitionID(nil), deployable...)
	r.mu.Unlock()
	if err == nil && r.bootstrapRecorder != nil {
		r.bootstrapRecorder.append(operationInitializeCheckpoints, appQName)
	}
	return deployable, failed, err
}

type mockInitialRecoveryAppStructsProvider struct {
	istructs.IAppStructsProvider
	recovery *mockInitialRecovery
}

func (p *mockInitialRecoveryAppStructsProvider) BuiltIn(appQName appdef.AppQName) (istructs.IAppStructs, error) {
	p.recovery.mu.Lock()
	appStructsGetErr := p.recovery.doFailAppStructsGet[appQName]
	p.recovery.mu.Unlock()
	if appStructsGetErr != nil {
		return nil, appStructsGetErr
	}
	appStructs, err := p.IAppStructsProvider.BuiltIn(appQName)
	if err == nil {
		p.recovery.mu.Lock()
		p.recovery.appQNamesByID[appStructs.ClusterAppID()] = appQName
		p.recovery.mu.Unlock()
	}
	return appStructs, err
}

func newPostWiredInterfacePtrs() btstrp.PostWireInterfacePtrs {
	return btstrp.PostWireInterfacePtrs{
		BlobberAppStorage: iblobstoragestg.BlobAppStoragePtr(new(istorage.IAppStorage)),
		RouterAppStorage:  dbcertcache.RouterAppStoragePtr(new(istorage.IAppStorage)),
		BlobHandler:       blobprocessor.IRequestHandlerPtr(new(blobprocessor.IRequestHandler)),
		RequestSender:     bus.IRequestSenderPtr(new(bus.IRequestSender)),
	}
}

func getTestVITCfg(numParts istructs.NumAppPartitions, numAppWS istructs.NumAppWorkspaces, opts ...it.VITConfigOptFunc) it.VITConfig {
	fs := fstest.MapFS{
		"app.vsql": &fstest.MapFile{
			Data: []byte(`APPLICATION app1();`),
		},
	}
	app1PackageFS := parser.PackageFS{
		Path: it.App1PkgPath,
		FS:   fs,
	}
	return it.NewOwnVITConfig(append([]it.VITConfigOptFunc{
		it.WithApp(istructs.AppQName_test1_app1, func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
			sysPkg := sysprovide.Provide(cfg)
			return builtinapps.Def{
				NumParts:         numParts,
				EnginePoolSize:   it.DefaultTestAppEnginesPool,
				NumAppWorkspaces: numAppWS,
				AppQName:         istructs.AppQName_test1_app1,
				Packages:         []parser.PackageFS{sysPkg, app1PackageFS},
			}
		}),
	}, opts...)...)
}

func TestDeployAppErrors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	sysToken, err := payloads.GetSystemPrincipalToken(vit.ITokens, istructs.AppQName_sys_cluster)
	require.NoError(err)

	t.Run("sys/cluster can not be deployed by c.cluster.DeployApp", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppQName":%q,"NumPartitions":1,"NumAppWorkspaces":1}}`, istructs.AppQName_sys_cluster)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID, "c.cluster.DeployApp", body,
			httpu.WithAuthorizeBy(sysToken), httpu.Expect400()).Println()
	})

	var test1App1DeploymentDescriptor appparts.AppDeploymentDescriptor
	for _, app := range vit.BuiltInAppsPackages {
		if app.Name == istructs.AppQName_test1_app1 {
			test1App1DeploymentDescriptor = app.AppDeploymentDescriptor
			break
		}
	}

	t.Run("409 conflict on try to deploy with different NumPartitions", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppQName":%q,"NumPartitions":%d,"NumAppWorkspaces":%d}}`,
			istructs.AppQName_test1_app1,
			test1App1DeploymentDescriptor.NumParts+1, test1App1DeploymentDescriptor.NumAppWorkspaces)
		resp := vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID, "c.cluster.DeployApp", body,
			httpu.WithAuthorizeBy(sysToken),
			httpu.Expect409(),
		)
		resp.Println()
		require.Empty(resp.NewIDs)
	})

	t.Run("409 conflict on try to deploy with different NumAppPartitions", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppQName":%q,"NumPartitions":%d,"NumAppWorkspaces":%d}}`,
			istructs.AppQName_test1_app1,
			test1App1DeploymentDescriptor.NumParts, test1App1DeploymentDescriptor.NumAppWorkspaces+1)
		resp := vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID, "c.cluster.DeployApp", body,
			httpu.WithAuthorizeBy(sysToken),
			httpu.Expect409(),
		)
		resp.Println()
		require.Empty(resp.NewIDs)
	})

	t.Run("400 bad request on wrong appQName", func(t *testing.T) {
		body := `{"args":{"AppQName":"wrong","NumPartitions":1,"NumAppWorkspaces":1}}`
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID, "c.cluster.DeployApp", body,
			httpu.WithAuthorizeBy(sysToken),
			httpu.Expect400(),
		).Println()
	})
}

func TestAppWSInitIndempotency(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	checkCDocsWSDesc(vit.VVMConfig, vit.VVM, require)

	// init app ws again (first is done on NewVIT()) -> expect no errors + assume next tests will work as well
	for _, app := range vit.BuiltInAppsPackages {
		as, err := vit.BuiltIn(app.Name)
		require.NoError(err)
		initedWSIDs, err := cluster.InitAppWSes(as, as.NumAppWorkspaces(), app.NumParts, istructs.UnixMilli(vit.Time.Now().UnixMilli()))
		require.NoError(err)
		require.Empty(initedWSIDs)
	}
}

func checkCDocsWSDesc(vvmCfg *vvm.VVMConfig, vvm *vvm.VVM, require *require.Assertions) {
	for appQName := range vvmCfg.VVMAppsBuilder {
		as, err := vvm.BuiltIn(appQName)
		require.NoError(err)
		for wsNum := 0; istructs.NumAppWorkspaces(wsNum) < as.NumAppWorkspaces(); wsNum++ {
			appWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
			existingCDocWSDesc, err := as.Records().GetSingleton(appWSID, appdef.QNameCDocWorkspaceDescriptor)
			require.NoError(err)
			require.Equal(appdef.QNameCDocWorkspaceDescriptor, existingCDocWSDesc.QName())
		}
	}
}

func TestVVMLaunch_VSQLCodeMismatch(t *testing.T) {
	launchAndRecover := func(t *testing.T, vsql string, registerExtraCmd bool) (panicErr error) {
		t.Helper()
		cfg := getMismatchVITCfg(vsql, registerExtraCmd)
		defer func() {
			if r := recover(); r != nil {
				panicErr, _ = r.(error)
			}
		}()
		vit := it.NewVIT(t, &cfg)
		vit.TearDown()
		return nil
	}

	t.Run("in vsql, not in code", func(t *testing.T) {
		require := require.New(t)
		vsql := `APPLICATION app1();
			ALTERABLE WORKSPACE test_wsWS (
				DESCRIPTOR test_ws (IntFld int32 NOT NULL);
				EXTENSION ENGINE BUILTIN (COMMAND missingCmd());
			);`
		err := launchAndRecover(t, vsql, false)
		require.ErrorIs(err, appparts.ErrDeployment, err)
		require.ErrorContains(err, "in vsql, not in code")
		require.ErrorContains(err, "missingCmd")
		log.Println(err)
	})

	t.Run("in code, not in vsql", func(t *testing.T) {
		require := require.New(t)
		vsql := `APPLICATION app1();
			ALTERABLE WORKSPACE test_wsWS (
				DESCRIPTOR test_ws (IntFld int32 NOT NULL);
			);`
		err := launchAndRecover(t, vsql, true)
		require.ErrorIs(err, appparts.ErrDeployment, err)
		require.ErrorContains(err, "in code, not in vsql")
		require.ErrorContains(err, "extraCmd")
		log.Println(err)
	})
}

func getMismatchVITCfg(vsql string, registerExtraCmd bool) it.VITConfig {
	const extraCmdName = "extraCmd"
	app1PackageFS := parser.PackageFS{
		Path: it.App1PkgPath,
		FS:   fstest.MapFS{"app.vsql": &fstest.MapFile{Data: []byte(vsql)}},
	}
	return it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app1, func(_ builtinapps.APIs, cfg *istructsmem.AppConfigType, _ extensionpoints.IExtensionPoint) builtinapps.Def {
			sysPkg := sysprovide.Provide(cfg)
			if registerExtraCmd {
				cfg.Resources.Add(istructsmem.NewCommandFunction(
					appdef.NewQName("app1pkg", extraCmdName), istructsmem.NullCommandExec))
			}
			return builtinapps.Def{
				AppDeploymentDescriptor: it.TestAppDeploymentDescriptor,
				AppQName:                istructs.AppQName_test1_app1,
				Packages:                []parser.PackageFS{sysPkg, app1PackageFS},
			}
		}),
	)
}
