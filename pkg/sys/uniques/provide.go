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
		view.Key().Partition().
			AddField(field_QName, appdef.DataKind_QName).
			AddField(field_ValuesHash, appdef.DataKind_int64)
		view.Key().ClustCols().
			AddBytesField(field_Values, appdef.DefaultFieldMaxLength)
		view.Value().
			AddRefField(field_ID, false) // true -> NullRecordID in required ref field error otherwise
	})
	cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameViewUniques,
			Func: provideUniquesProjectorFunc(appDefBuilder),
		}
	})
	cfg.AddCUDValidators(istructs.CUDValidator{
		MatchFunc: func(qName appdef.QName) bool {
			if uniques, ok := appDefBuilder.Def(qName).(appdef.IUniques); ok {
				return uniques.UniqueField() != nil
			}
			return false
		},
		Validate: provideCUDUniqueUpdateDenyValidator(),
	})
	cfg.AddEventValidators(provideEventUniqueValidator())
}
