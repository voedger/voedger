/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestAppTTLStorage_BasicUsage(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	require := require.New(t)

	t.Run("Put and Get", func(t *testing.T) {
		body := `{"args":{"Operation":"Put","Key":"test-key-1","Value":"test-value-1","TTLSeconds":3600}}`
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.True(resp.CmdResult["Ok"].(bool))

		body = `{"args":{"Key":"test-key-1"},"elements":[{"fields":["Value","Exists"]}]}`
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.Equal("test-value-1", resp.SectionRow()[0])
		require.True(resp.SectionRow()[1].(bool))
	})

	t.Run("Put when key exists returns false", func(t *testing.T) {
		body := `{"args":{"Operation":"Put","Key":"test-key-2","Value":"first-value","TTLSeconds":3600}}`
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.True(resp.CmdResult["Ok"].(bool))

		body = `{"args":{"Operation":"Put","Key":"test-key-2","Value":"second-value","TTLSeconds":3600}}`
		resp = vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.False(resp.CmdResult["Ok"].(bool))

		body = `{"args":{"Key":"test-key-2"},"elements":[{"fields":["Value","Exists"]}]}`
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.Equal("first-value", resp.SectionRow()[0])
	})

	t.Run("Get non-existing key", func(t *testing.T) {
		body := `{"args":{"Key":"non-existing-key"},"elements":[{"fields":["Value","Exists"]}]}`
		resp := vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.Empty(resp.SectionRow()[0])
		require.False(resp.SectionRow()[1].(bool))
	})

	t.Run("TTL expiration", func(t *testing.T) {
		key := vit.NextName()

		body := fmt.Sprintf(`{"args":{"Operation":"Put","Key":%q,"Value":"expiring-value","TTLSeconds":60}}`, key)
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.True(resp.CmdResult["Ok"].(bool))

		body = fmt.Sprintf(`{"args":{"Key":%q},"elements":[{"fields":["Value","Exists"]}]}`, key)
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.True(resp.SectionRow()[1].(bool))

		vit.TimeAdd(61 * time.Second)
		ws = vit.WS(istructs.AppQName_test1_app1, "test_ws")

		body = fmt.Sprintf(`{"args":{"Key":%q},"elements":[{"fields":["Value","Exists"]}]}`, key)
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.False(resp.SectionRow()[1].(bool))
	})
}

func TestAppTTLStorage_CompareAndSwap(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	require := require.New(t)

	body := `{"args":{"Operation":"Put","Key":"cas-key","Value":"initial","TTLSeconds":3600}}`
	resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
	require.True(resp.CmdResult["Ok"].(bool))

	t.Run("swap with correct expected value", func(t *testing.T) {
		body := `{"args":{"Operation":"CompareAndSwap","Key":"cas-key","ExpectedValue":"initial","Value":"updated","TTLSeconds":3600}}`
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.True(resp.CmdResult["Ok"].(bool))

		body = `{"args":{"Key":"cas-key"},"elements":[{"fields":["Value","Exists"]}]}`
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.Equal("updated", resp.SectionRow()[0])
	})

	t.Run("swap with wrong expected value", func(t *testing.T) {
		body := `{"args":{"Operation":"CompareAndSwap","Key":"cas-key","ExpectedValue":"wrong","Value":"new-value","TTLSeconds":3600}}`
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.False(resp.CmdResult["Ok"].(bool))

		body = `{"args":{"Key":"cas-key"},"elements":[{"fields":["Value","Exists"]}]}`
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.Equal("updated", resp.SectionRow()[0])
	})

	t.Run("swap fails after expiration", func(t *testing.T) {
		key := vit.NextName()

		body := fmt.Sprintf(`{"args":{"Operation":"Put","Key":%q,"Value":"expiring","TTLSeconds":60}}`, key)
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.True(resp.CmdResult["Ok"].(bool))

		vit.TimeAdd(61 * time.Second)
		ws = vit.WS(istructs.AppQName_test1_app1, "test_ws")

		body = fmt.Sprintf(`{"args":{"Operation":"CompareAndSwap","Key":%q,"ExpectedValue":"expiring","Value":"new","TTLSeconds":60}}`, key)
		resp = vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.False(resp.CmdResult["Ok"].(bool))
	})
}

func TestAppTTLStorage_CompareAndDelete(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	require := require.New(t)

	body := `{"args":{"Operation":"Put","Key":"cad-key","Value":"to-delete","TTLSeconds":3600}}`
	resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
	require.True(resp.CmdResult["Ok"].(bool))

	t.Run("delete with wrong expected value", func(t *testing.T) {
		body := `{"args":{"Operation":"CompareAndDelete","Key":"cad-key","ExpectedValue":"wrong"}}`
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.False(resp.CmdResult["Ok"].(bool))

		body = `{"args":{"Key":"cad-key"},"elements":[{"fields":["Value","Exists"]}]}`
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.True(resp.SectionRow()[1].(bool))
	})

	t.Run("delete with correct expected value", func(t *testing.T) {
		body := `{"args":{"Operation":"CompareAndDelete","Key":"cad-key","ExpectedValue":"to-delete"}}`
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.True(resp.CmdResult["Ok"].(bool))

		body = `{"args":{"Key":"cad-key"},"elements":[{"fields":["Value","Exists"]}]}`
		resp = vit.PostWS(ws, "q.app1pkg.TTLGetQry", body)
		require.False(resp.SectionRow()[1].(bool))
	})

	t.Run("delete fails after expiration", func(t *testing.T) {
		key := vit.NextName()

		body := fmt.Sprintf(`{"args":{"Operation":"Put","Key":%q,"Value":"expiring","TTLSeconds":60}}`, key)
		resp := vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.True(resp.CmdResult["Ok"].(bool))

		vit.TimeAdd(61 * time.Second)
		ws = vit.WS(istructs.AppQName_test1_app1, "test_ws")

		body = fmt.Sprintf(`{"args":{"Operation":"CompareAndDelete","Key":%q,"ExpectedValue":"expiring"}}`, key)
		resp = vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body)
		require.False(resp.CmdResult["Ok"].(bool))
	})
}

func TestAppTTLStorage_ValidationErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("empty key returns 400", func(t *testing.T) {
		// empty key validation happens at VSQL level before reaching the command handler
		body := `{"args":{"Operation":"Put","Key":"","Value":"value","TTLSeconds":3600}}`
		vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body, it.Expect400("field is empty"))
	})

	t.Run("invalid TTL returns 400", func(t *testing.T) {
		body := `{"args":{"Operation":"Put","Key":"valid-key","Value":"value","TTLSeconds":0}}`
		vit.PostWS(ws, "c.app1pkg.TTLStorageCmd", body, it.Expect400("TTL must be between"))
	})

	t.Run("query with empty key returns 400", func(t *testing.T) {
		// empty key validation happens at VSQL level before reaching the query handler
		body := `{"args":{"Key":""},"elements":[{"fields":["Value","Exists"]}]}`
		vit.PostWS(ws, "q.app1pkg.TTLGetQry", body, it.Expect400("field is empty"))
	})
}
