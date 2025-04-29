/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	it "github.com/voedger/voedger/pkg/vit"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestRecovery(t *testing.T) {
	require := require.New(t)
	keyspaceSuffix := uuid.NewString()
	storageFactory := mem.Provide(coreutils.MockTime)
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
				// 1st VVM launch - use RegisterFactor 2 (5_000_000_000) to issue big IDs
				var err error
				sharedStorageFactory, err = cfg.StorageFactory()
				require.NoError(err)
				cfg.KeyspaceNameSuffix = keyspaceSuffix
				cfg.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
					return sharedStorageFactory, nil
				}
				cfg.IDGeneratorFactory = func() istructs.IIDGenerator {
					return &idGenRegister2{
						IIDGenerator: istructsmem.NewIDGenerator(),
					}
				}
			case 2:
				// 2nd VVM launch -> use the normal IDGenerator with RegisterFactor 1
				// to check if new IDs would start from RegisterFactor 2 after recovery
				cfg.IDGeneratorFactory = istructsmem.NewIDGenerator
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
	ws = vit.WS(istructs.AppQName_test1_app1, "test_ws")
	body = `{"cuds": [
		{"fields":{"sys.ID": 1,"sys.QName": "app1pkg.Root", "FldRoot": 2}},
		{"fields":{"sys.ID": 2,"sys.QName": "app1pkg.Nested", "sys.ParentID":1,"sys.Container": "Nested","FldNested":3}},
		{"fields":{"sys.ID": 3,"sys.QName": "app1pkg.Third", "Fld1": 42,"sys.ParentID":2,"sys.Container": "Third"}}
	]}`
	resp = vit.PostWS(ws, "c.sys.CUD", body)
	resp.Println()

	_ = storageFactory
}

type idGenRegister2 struct {
	istructs.IIDGenerator
}

func (idGen *idGenRegister2) NextID(rawID istructs.RecordID) (storageID istructs.RecordID, err error) {
	storageID, err = idGen.IIDGenerator.NextID(rawID)
	return storageID + 5_000_000_000, err
}
