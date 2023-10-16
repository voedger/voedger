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

var QNameViewUniques = appdef.NewQName(appdef.SysPackage, "Uniques")

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {

	projectors.ProvideViewDef(appDefBuilder, QNameViewUniques, func(view appdef.IViewBuilder) {
		view.KeyBuilder().PartKeyBuilder().
			AddField(field_QName, appdef.DataKind_QName).
			AddField(field_ValuesHash, appdef.DataKind_int64)
		view.KeyBuilder().ClustColsBuilder().
			AddBytesField(field_Values, appdef.DefaultFieldMaxLength)
		view.ValueBuilder().
			AddRefField(field_ID, false) // true -> NullRecordID in required ref field error otherwise
	})
	cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameViewUniques,
			Func: provideUniquesProjectorFunc(appDefBuilder),
		}
	})
	cfg.AddEventValidators(provideEventUniqueValidator())
}
