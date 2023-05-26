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
	appDef := New()

	saleParamsDef := appDef.AddODoc(NewQName("test", "Sale"))
	saleParamsDef.
		AddField("Buyer", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)
	saleParamsDef.
		AddContainer("Basket", NewQName("test", "Basket"), 1, 1)

	basketDef := appDef.AddORecord(NewQName("test", "Basket"))
	basketDef.AddContainer("Good", NewQName("test", "Good"), 0, Occurs_Unbounded)

	goodDef := appDef.AddORecord(NewQName("test", "Good"))
	goodDef.
		AddField("Name", DataKind_string, true).
		AddField("Code", DataKind_int64, true).
		AddField("Weight", DataKind_float64, false)

	saleSecureParamsDef := appDef.AddObject(NewQName("test", "saleSecureArgs"))
	saleSecureParamsDef.
		AddField("password", DataKind_string, true)

	docDef := appDef.AddCDoc(NewQName("test", "photos"))
	docDef.
		AddField("Buyer", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)

	viewDef := appDef.AddView(NewQName("test", "viewBuyerByHeight"))
	viewDef.
		AddPartField("Height", DataKind_float32).
		AddClustColumn("Buyer", DataKind_string).
		AddValueField("BuyerID", DataKind_RecordID, true)

	result, err := appDef.Build()

	t.Run("test results", func(t *testing.T) {
		require := require.New(t)
		require.NoError(err)
		require.NotNil(result)
	})

}
