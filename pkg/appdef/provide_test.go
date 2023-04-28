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

	saleParamsSchema := appDef.Add(NewQName("test", "Sale"), DefKind_ODoc)
	saleParamsSchema.
		AddField("Buyer", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false).
		AddContainer("Basket", NewQName("test", "Basket"), 1, 1)

	basketSchema := appDef.Add(NewQName("test", "Basket"), DefKind_ORecord)
	basketSchema.AddContainer("Good", NewQName("test", "Good"), 0, Occurs_Unbounded)

	goodSchema := appDef.Add(NewQName("test", "Good"), DefKind_ORecord)
	goodSchema.
		AddField("Name", DataKind_string, true).
		AddField("Code", DataKind_int64, true).
		AddField("Weight", DataKind_float64, false)

	saleSecurParamsSchema := appDef.Add(NewQName("test", "saleSecureArgs"), DefKind_Object)
	saleSecurParamsSchema.
		AddField("password", DataKind_string, true)

	docSchema := appDef.Add(NewQName("test", "photos"), DefKind_CDoc)
	docSchema.
		AddField("Buyer", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)

	viewSchema := appDef.AddView(NewQName("test", "viewBuyerByHeight"))
	viewSchema.
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
