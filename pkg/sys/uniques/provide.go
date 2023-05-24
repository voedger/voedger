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
	projectors.ProvideViewDef(appDefBuilder, QNameViewUniques, func(b appdef.IViewBuilder) {
		b.AddPartField(field_QName, appdef.DataKind_QName).
			AddPartField(field_ValuesHash, appdef.DataKind_int64).
			AddClustColumn(field_Values, appdef.DataKind_bytes).
			AddValueField(field_ID, appdef.DataKind_RecordID, true)
	})
	cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameViewUniques,
			Func: provideUniquesProjectorFunc(cfg.Uniques, appDefBuilder),
		}
	})
	cfg.AddCUDValidators(istructs.CUDValidator{
		MatchFunc: func(qName appdef.QName) bool {
			return len(cfg.Uniques.GetAll(qName)) > 0
		},
		Validate: provideCUDUniqueUpdateDenyValidator(cfg.Uniques),
	})
	cfg.AddEventValidators(provideEventUniqueValidator(cfg.Uniques))
}
