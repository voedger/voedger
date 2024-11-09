/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage(t *testing.T) {
	adb := New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(NewQName("test", "workspace"))

	saleParamsDoc := wsb.AddODoc(NewQName("test", "Sale"))
	saleParamsDoc.
		AddField("Buyer", DataKind_string, true, MaxLen(100)).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)
	saleParamsDoc.
		AddContainer("Basket", NewQName("test", "Basket"), 1, 1)

	basketRec := wsb.AddORecord(NewQName("test", "Basket"))
	basketRec.AddContainer("Good", NewQName("test", "Good"), 0, Occurs_Unbounded)

	goodRec := wsb.AddORecord(NewQName("test", "Good"))
	goodRec.
		AddField("Name", DataKind_string, true).
		AddField("Code", DataKind_int64, true).
		AddField("Weight", DataKind_float64, false)

	saleSecureParamsObj := wsb.AddObject(NewQName("test", "saleSecureArgs"))
	saleSecureParamsObj.
		AddField("password", DataKind_string, true)

	docName := NewQName("test", "photos")
	photosDoc := wsb.AddCDoc(docName)
	photosDoc.
		AddField("Buyer", DataKind_string, true, MaxLen(100)).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)

	buyerView := wsb.AddView(NewQName("test", "viewBuyerByHeight"))
	buyerView.Key().PartKey().AddField("Height", DataKind_float32)
	buyerView.Key().ClustCols().AddField("Buyer", DataKind_string, MaxLen(100))
	buyerView.Value().AddRefField("BuyerID", true, docName)

	buyerObjName := NewQName("test", "buyer")
	buyerObj := wsb.AddObject(buyerObjName)
	buyerObj.
		AddField("Name", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("isHuman", DataKind_bool, false)

	newBuyerCmd := wsb.AddCommand(NewQName("test", "cmdNewBuyer"))
	newBuyerCmd.SetParam(buyerObjName)

	appDef, err := adb.Build()

	t.Run("test results", func(t *testing.T) {
		require := require.New(t)
		require.NoError(err)
		require.NotNil(appDef)
	})
}
