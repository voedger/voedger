/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/projectors"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {

	projectors.ProvideViewDef(appDefBuilder, qNameViewUniques, func(view appdef.IViewBuilder) {
		view.KeyBuilder().PartKeyBuilder().
			AddField(field_QName, appdef.DataKind_QName).
			AddField(field_ValuesHash, appdef.DataKind_int64)
		view.KeyBuilder().ClustColsBuilder().
			AddField(field_Values, appdef.DataKind_bytes, appdef.MaxLen(appdef.DefaultFieldMaxLength))
		view.ValueBuilder().
			AddRefField(field_ID, false) // true -> NullRecordID in required ref field error otherwise
	})
	cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameApplyUniques,
			Func: provideApplyUniques(appDefBuilder),
		}
	})
	cfg.AddEventValidators(provideEventUniqueValidator())
}
