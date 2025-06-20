/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestCorrectIDsIssueAfterRecovery(t *testing.T) {
	require := require.New(t)
	keyspaceSuffix := uuid.NewString()
	var sharedStorageFactory istorage.IAppStorageFactory
	counter := 1
	cfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app1, it.ProvideApp1,
			it.WithWorkspaceTemplate(it.QNameApp1_TestWSKind, "test_template", sys_test_template.TestTemplateFS),
			it.WithUserLogin("login", "pwd"),
			it.WithChildWorkspace(it.QNameApp1_TestWSKind, "test_ws", "test_template", "", "login", map[string]interface{}{"IntFld": 42}),
		),
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			switch counter {
			case 1:
				// 1st VVM launch
				var err error
				sharedStorageFactory, err = cfg.StorageFactory()
				require.NoError(err)
				cfg.KeyspaceNameSuffix = keyspaceSuffix
				cfg.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
					return sharedStorageFactory, nil
				}
			case 2:
				// 2nd VVM launch - the IDGenerator must be updated on recovery
				cfg.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
					return sharedStorageFactory, nil
				}
				cfg.KeyspaceNameSuffix = keyspaceSuffix
			}
		}),
	)
	vit := it.NewVIT(t, &cfg)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `{"args":{"sys.ID": 1,"orecord1":[{"sys.ID":2,"sys.ParentID":1,"orecord2":[{"sys.ID":3,"sys.ParentID":2}]}]}}`
	resp := vit.PostWS(ws, "c.app1pkg.CmdODocOne", body)
	resp.Println()

	body = `{"cuds": [
		{"fields":{"sys.ID": 1,"sys.QName": "app1pkg.Root", "FldRoot": 2}},
		{"fields":{"sys.ID": 2,"sys.QName": "app1pkg.Nested", "sys.ParentID":1,"sys.Container": "Nested","FldNested":3}},
		{"fields":{"sys.ID": 3,"sys.QName": "app1pkg.Third", "Fld1": 42,"sys.ParentID":2,"sys.Container": "Third"}}
	]}`
	resp = vit.PostWS(ws, "c.sys.CUD", body)
	resp.Println()

	vit.TearDown()

	// 2nd launch - check if new ids issued correctly
	counter++
	vit = it.NewVIT(t, &cfg)
	defer vit.TearDown()
	ws = vit.WS(istructs.AppQName_test1_app1, "test_ws")
	body = `{"cuds": [
		{"fields":{"sys.ID": 1,"sys.QName": "app1pkg.Root", "FldRoot": 2}},
		{"fields":{"sys.ID": 2,"sys.QName": "app1pkg.Nested", "sys.ParentID":1,"sys.Container": "Nested","FldNested":3}},
		{"fields":{"sys.ID": 3,"sys.QName": "app1pkg.Third", "Fld1": 42,"sys.ParentID":2,"sys.Container": "Third"}}
	]}`
	resp = vit.PostWS(ws, "c.sys.CUD", body)
	resp.Println()
}
