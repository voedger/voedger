/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"testing"

	"github.com/untillpro/voedger/pkg/istructs"
)

func TestBasicUsage(t *testing.T) {
	schemas := NewSchemaCache()

	saleParamsSchema := schemas.Add(istructs.NewQName("test", "Sale"), istructs.SchemaKind_ODoc)
	saleParamsSchema.
		AddField("Buyer", istructs.DataKind_string, true).
		AddField("Age", istructs.DataKind_int32, false).
		AddField("Height", istructs.DataKind_float32, false).
		AddField("isHuman", istructs.DataKind_bool, false).
		AddField("Photo", istructs.DataKind_bytes, false).
		AddContainer("Basket", istructs.NewQName("test", "Basket"), 1, 1)

	basketSchema := schemas.Add(istructs.NewQName("test", "Basket"), istructs.SchemaKind_ORecord)
	basketSchema.AddContainer("Good", istructs.NewQName("test", "Good"), 0, istructs.ContainerOccurs_Unbounded)

	goodSchema := schemas.Add(istructs.NewQName("test", "Good"), istructs.SchemaKind_ORecord)
	goodSchema.
		AddField("Name", istructs.DataKind_string, true).
		AddField("Code", istructs.DataKind_int64, true).
		AddField("Weight", istructs.DataKind_float64, false)

	saleSecurParamsSchema := schemas.Add(istructs.NewQName("test", "saleSecureArgs"), istructs.SchemaKind_Object)
	saleSecurParamsSchema.
		AddField("password", istructs.DataKind_string, true)

	docSchema := schemas.Add(istructs.NewQName("test", "photos"), istructs.SchemaKind_CDoc)
	docSchema.
		AddField("Buyer", istructs.DataKind_string, true).
		AddField("Age", istructs.DataKind_int32, false).
		AddField("Height", istructs.DataKind_float32, false).
		AddField("isHuman", istructs.DataKind_bool, false).
		AddField("Photo", istructs.DataKind_bytes, false)

	viewSchema := schemas.AddView(istructs.NewQName("test", "viewBuyerByHeight"))
	viewSchema.
		AddPartField("Height", istructs.DataKind_float32).
		AddClustColumn("Buyer", istructs.DataKind_string).
		AddValueField("BuyerID", istructs.DataKind_RecordID, true)

	if err := schemas.ValidateSchemas(); err != nil {
		panic(err)
	}
}
