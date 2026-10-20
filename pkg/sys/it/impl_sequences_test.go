/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestSequenceAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Run("sequences: scn: PLog offsets advance within an application partition", func(t *testing.T) {
		vit, owner := newSequenceVIT(t, 2)
		defer vit.TearDown()

		// Given workspace "Orders" and workspace "Billing" belong to application partition 1
		vit.CreateWorkspace(it.SimpleWSParams("partition-zero-1"), owner)
		orders := vit.CreateWorkspace(it.SimpleWSParams("Orders"), owner)
		vit.CreateWorkspace(it.SimpleWSParams("partition-zero-2"), owner)
		billing := vit.CreateWorkspace(it.SimpleWSParams("Billing"), owner)
		ordersPartition, err := vit.IAppPartitions.AppWorkspacePartitionID(orders.AppQName(), orders.WSID)
		require.NoError(t, err)
		billingPartition, err := vit.IAppPartitions.AppWorkspacePartitionID(billing.AppQName(), billing.WSID)
		require.NoError(t, err)
		require.Equal(t, istructs.PartitionID(1), ordersPartition)
		require.Equal(t, ordersPartition, billingPartition)

		// And the latest PLog event in partition 1 has offset 40
		fillPLogWithNonIDBearingEvents(t, vit, orders.AppQName(), ordersPartition, owner, 40)
		require.Equal(t, istructs.Offset(40), getLastPLogOffset(t, vit, orders.AppQName(), ordersPartition))

		// When VAOperator successfully submits a command in workspace "Orders"
		vit.PostWS(orders, nonIDBearingCommand, "{}")
		ordersOffset := getLastPLogOffset(t, vit, orders.AppQName(), ordersPartition)

		// And VAOperator successfully submits a command in workspace "Billing"
		vit.PostWS(billing, nonIDBearingCommand, "{}")
		billingOffset := getLastPLogOffset(t, vit, billing.AppQName(), billingPartition)

		// Then the command in workspace "Orders" is persisted as PLog event 41
		require.Equal(t, istructs.Offset(41), ordersOffset)
		// And the command in workspace "Billing" is persisted as PLog event 42
		require.Equal(t, istructs.Offset(42), billingOffset)
	})

	t.Run("sequences: scn: WLog offsets advance independently in each workspace", func(t *testing.T) {
		vit, owner := newSequenceVIT(t, 1)
		defer vit.TearDown()
		orders := vit.CreateWorkspace(it.SimpleWSParams("Orders"), owner)
		billing := vit.CreateWorkspace(it.SimpleWSParams("Billing"), owner)

		// Given the WLog of workspace "Orders" ends at offset 7
		fillWLogWithNonIDBearingEvents(t, vit, orders, 7)
		// And the WLog of workspace "Billing" ends at offset 12
		fillWLogWithNonIDBearingEvents(t, vit, billing, 12)

		// When VAOperator successfully submits one command in each workspace
		ordersResponse := vit.PostWS(orders, nonIDBearingCommand, "{}")
		billingResponse := vit.PostWS(billing, nonIDBearingCommand, "{}")

		// Then the command in workspace "Orders" is persisted at WLog offset 8
		require.Equal(t, istructs.Offset(8), ordersResponse.CurrentWLogOffset)
		// And the command in workspace "Billing" is persisted at WLog offset 13
		require.Equal(t, istructs.Offset(13), billingResponse.CurrentWLogOffset)
	})

	t.Run("sequences: scn: A command receives consecutive ordinary record IDs", func(t *testing.T) {
		vit, owner := newSequenceVIT(t, 1)
		defer vit.TearDown()
		orders := vit.CreateWorkspace(it.SimpleWSParams("Orders"), owner)

		// Given the next ordinary record ID in workspace "Orders" is 200001
		require.Equal(t, istructs.RecordID(200001), istructs.FirstUserRecordID)

		// When VAOperator successfully creates three ordinary records in one command
		response := vit.PostWS(orders, "c.sys.CUD", docWithNested)

		// Then the command returns record IDs 200001 through 200003 in creation order
		require.Equal(t, istructs.RecordID(200001), response.NewIDs["1"])
		require.Equal(t, istructs.RecordID(200002), response.NewIDs["2"])
		require.Equal(t, istructs.RecordID(200003), response.NewIDs["3"])

		// When VAOperator successfully creates another ordinary record in workspace "Orders"
		response = vit.PostWS(orders, "c.sys.CUD", singleDoc)

		// Then the command returns record ID 200004
		require.Equal(t, istructs.RecordID(200004), response.NewIDs["1"])
	})

	t.Run("sequences: scn: Ordinary record-ID sequences are independent between workspaces", func(t *testing.T) {
		vit, owner := newSequenceVIT(t, 1)
		defer vit.TearDown()

		// Given workspace "Orders" and workspace "Billing" have no ordinary records
		orders := vit.CreateWorkspace(it.SimpleWSParams("Orders"), owner)
		billing := vit.CreateWorkspace(it.SimpleWSParams("Billing"), owner)

		// When VAOperator successfully creates one ordinary record in each workspace
		ordersResponse := vit.PostWS(orders, "c.sys.CUD", singleDoc)
		billingResponse := vit.PostWS(billing, "c.sys.CUD", singleDoc)

		// Then each command returns record ID 200001
		require.Equal(t, istructs.RecordID(200001), ordersResponse.NewIDs["1"])
		require.Equal(t, istructs.RecordID(200001), billingResponse.NewIDs["1"])
	})

	t.Run("sequences: scn: A singleton uses its assigned ID without consuming an ordinary record ID", func(t *testing.T) {
		vit, owner := newSequenceVIT(t, 1)
		defer vit.TearDown()
		orders := vit.CreateWorkspace(it.SimpleWSParams("Orders"), owner)
		appStructs, err := vit.IAppStructsProvider.BuiltIn(orders.AppQName())
		require.NoError(t, err)

		// Given singleton "Configuration" has application-defined record ID 65536
		configurationID, err := appStructs.Records().GetSingletonID(appdef.NewQName("app1pkg", "Config"))
		require.NoError(t, err)
		require.Equal(t, istructs.RecordID(65536), configurationID)
		// And the next ordinary record ID in workspace "Orders" is 200001
		require.Equal(t, istructs.RecordID(200001), istructs.FirstUserRecordID)

		// When VAOperator successfully creates the "Configuration" singleton
		response := vit.PostWS(orders, "c.sys.CUD", singletonCUD)

		// Then the stored singleton has record ID 65536
		configuration, err := appStructs.Records().GetSingleton(orders.WSID, appdef.NewQName("app1pkg", "Config"))
		require.NoError(t, err)
		require.Equal(t, istructs.RecordID(65536), configuration.ID())
		// And the command returns no generated ordinary record ID
		require.Empty(t, response.NewIDs)

		// When VAOperator successfully creates an ordinary record in workspace "Orders"
		response = vit.PostWS(orders, "c.sys.CUD", singleDoc)

		// Then the command returns record ID 200001
		require.Equal(t, istructs.RecordID(200001), response.NewIDs["1"])
	})

	t.Run("sequences: scn: Workspace-ID sequences are scoped to an application", func(t *testing.T) {
		cfg := it.NewOwnVITConfig(
			it.WithApp(istructs.AppQName_test1_app1, provideTestApp),
			it.WithApp(istructs.AppQName_test1_app2, it.ProvideApp2),
		)
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()

		// Given application "Sales" and application "Support" have no user workspaces
		salesOwner := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
		supportOwner := vit.GetSystemPrincipal(istructs.AppQName_test1_app2)

		// When VAOperator creates workspaces "Orders" and "Billing" in application "Sales"
		orders := vit.CreateWorkspace(it.SimpleWSParams("Orders"), salesOwner)
		billing := vit.CreateWorkspace(it.SimpleWSParams("Billing"), salesOwner)
		// And VAOperator creates workspace "Helpdesk" in application "Support"
		helpdeskParams := it.WSParams{
			Name:         "Helpdesk",
			Kind:         appdef.NewQName("app2pkg", "test_ws"),
			ClusterID:    istructs.CurrentClusterID(),
			InitDataJSON: `{"IntFld": 42}`,
		}
		helpdesk := vit.CreateWorkspace(helpdeskParams, supportOwner)

		// Then workspace "Orders" has base WSID 131071
		require.Equal(t, istructs.WSID(131071), orders.WSID.BaseWSID())
		// And workspace "Billing" has base WSID 131072
		require.Equal(t, istructs.WSID(131072), billing.WSID.BaseWSID())
		// And workspace "Helpdesk" has base WSID 131071
		require.Equal(t, istructs.WSID(131071), helpdesk.WSID.BaseWSID())
	})
}

func newSequenceVIT(t *testing.T, numPartitions istructs.NumAppPartitions) (*it.VIT, *it.Principal) {
	t.Helper()
	appQName := istructs.AppQName_test1_app1
	cfg := it.NewOwnVITConfig(it.WithApp(appQName, provideTestAppWithPartitions(numPartitions),
		it.WithUserLogin(recoveryTestLoginName, recoveryTestLoginPassword),
	))
	vit := it.NewVIT(t, &cfg)
	return vit, vit.GetPrincipal(appQName, recoveryTestLoginName)
}
