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

	saleParamsDef := appDef.Add(NewQName("test", "Sale"), DefKind_ODoc)
	saleParamsDef.
		AddField("Buyer", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false).
		AddContainer("Basket", NewQName("test", "Basket"), 1, 1)

	basketDef := appDef.Add(NewQName("test", "Basket"), DefKind_ORecord)
	basketDef.AddContainer("Good", NewQName("test", "Good"), 0, Occurs_Unbounded)

	goodDef := appDef.Add(NewQName("test", "Good"), DefKind_ORecord)
	goodDef.
		AddField("Name", DataKind_string, true).
		AddField("Code", DataKind_int64, true).
		AddField("Weight", DataKind_float64, false)

	saleSecurParamsDef := appDef.Add(NewQName("test", "saleSecureArgs"), DefKind_Object)
	saleSecurParamsDef.
		AddField("password", DataKind_string, true)

	docDef := appDef.Add(NewQName("test", "photos"), DefKind_CDoc)
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
