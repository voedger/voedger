/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage(t *testing.T) {
	schemas := NewSchemaCache()

	saleParamsSchema := schemas.Add(NewQName("test", "Sale"), SchemaKind_ODoc)
	saleParamsSchema.
		AddField("Buyer", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false).
		AddContainer("Basket", NewQName("test", "Basket"), 1, 1)

	basketSchema := schemas.Add(NewQName("test", "Basket"), SchemaKind_ORecord)
	basketSchema.AddContainer("Good", NewQName("test", "Good"), 0, Occurs_Unbounded)

	goodSchema := schemas.Add(NewQName("test", "Good"), SchemaKind_ORecord)
	goodSchema.
		AddField("Name", DataKind_string, true).
		AddField("Code", DataKind_int64, true).
		AddField("Weight", DataKind_float64, false)

	saleSecurParamsSchema := schemas.Add(NewQName("test", "saleSecureArgs"), SchemaKind_Object)
	saleSecurParamsSchema.
		AddField("password", DataKind_string, true)

	docSchema := schemas.Add(NewQName("test", "photos"), SchemaKind_CDoc)
	docSchema.
		AddField("Buyer", DataKind_string, true).
		AddField("Age", DataKind_int32, false).
		AddField("Height", DataKind_float32, false).
		AddField("isHuman", DataKind_bool, false).
		AddField("Photo", DataKind_bytes, false)

	viewSchema := schemas.AddView(NewQName("test", "viewBuyerByHeight"))
	viewSchema.
		AddPartField("Height", DataKind_float32).
		AddClustColumn("Buyer", DataKind_string).
		AddValueField("BuyerID", DataKind_RecordID, true)

	result, err := schemas.Build()

	t.Run("test results", func(t *testing.T) {
		require := require.New(t)
		require.NoError(err)
		require.NotNil(result)
	})

}
