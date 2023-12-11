/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package sys_it

import (
	"testing"

	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_RenameQName(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	// ensure untill.category is not renamed yet
	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)

	bodyRename := `{"args":{"ExistingQName":"app1pkg.category","NewQName":"app1pkg.categorynew"}}`
	vit.PostWS(ws, "c.sys.RenameQName", bodyRename)

	// need to restart the server
	// body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.categorynew","name":"Awesome food"}}]}`
	// vit.PostWS(ws, "c.sys.CUD", body)

	// hit.PostApp(istructs.AppQName_test1_app1, ws.Owner.PseudoProfileWSID, "c.sys.RenameQName", body, utils.WithAuthorizeBy(sysToken))

	// // ensure c.sys.CUD is not renamed yet
	// body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.category","name":"Awesome food"}}]}`
	// hit.PostWS(ws, "c.sys.CUDnew", body, utils.Expect400())

	// sysToken, err := payloads.GetSystemPrincipalToken(hit.ITokens, istructs.AppQName_test1_app1)
	// require.NoError(t, err)
	// hit.PostApp(istructs.AppQName_test1_app1, ws.Owner.PseudoProfileWSID, "c.sys.CUD", body, utils.WithAuthorizeBy(sysToken))

	// // rename command
	// // body = `{"args":{"ExistingQName":"sys.CUD","NewQName":"sys.CUDnew"}}`
	// body = `{"args":{"ExistingQName":"untill.category","NewQName":"untill.categorynew"}}`
	// hit.PostApp(istructs.AppQName_test1_app1, ws.Owner.PseudoProfileWSID, "c.sys.RenameQName", body, utils.WithAuthorizeBy(sysToken))

	// // try to call the command by the new QName
	// body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.categorynew","name":"Awesome food"}}]}`
	// hit.PostWS(ws, "c.sys.CUD", body)
}
