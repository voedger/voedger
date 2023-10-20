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

	saleParamsDoc := adb.AddODoc(NewQName("test", "Sale"))
	saleParamsDoc.
		AddStringField("Buyer", true, MaxLen(100)).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)
	saleParamsDoc.
		AddContainer("Basket", NewQName("test", "Basket"), 1, 1)

	basketRec := adb.AddORecord(NewQName("test", "Basket"))
	basketRec.AddContainer("Good", NewQName("test", "Good"), 0, Occurs_Unbounded)

	goodRec := adb.AddORecord(NewQName("test", "Good"))
	goodRec.
		AddField("Name", DataKind_string, true).
		AddField("Code", DataKind_int64, true).
		AddField("Weight", DataKind_float64, false)

	saleSecureParamsObj := adb.AddObject(NewQName("test", "saleSecureArgs"))
	saleSecureParamsObj.
		AddField("password", DataKind_string, true)

	docName := NewQName("test", "photos")
	photosDoc := adb.AddCDoc(docName)
	photosDoc.
		AddStringField("Buyer", true, MaxLen(100)).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)

	buyerView := adb.AddView(NewQName("test", "viewBuyerByHeight"))
	buyerView.KeyBuilder().PartKeyBuilder().AddField("Height", DataKind_float32)
	buyerView.KeyBuilder().ClustColsBuilder().AddStringField("Buyer", 100)
	buyerView.ValueBuilder().AddRefField("BuyerID", true, docName)

	buyerObj := adb.AddObject(NewQName("test", "buyer"))
	buyerObj.
		AddStringField("Name", true).
		AddField("Age", DataKind_int32, false).
		AddField("isHuman", DataKind_bool, false)

	newBuyerCmd := adb.AddCommand(NewQName("test", "cmdNewBuyer"))
	newBuyerCmd.SetParam(buyerObj.QName())
	newBuyerCmd.SetExtension("newBuyer", ExtensionEngineKind_BuiltIn)

	appDef, err := adb.Build()

	t.Run("test results", func(t *testing.T) {
		require := require.New(t)
		require.NoError(err)
		require.NotNil(appDef)
	})

}
