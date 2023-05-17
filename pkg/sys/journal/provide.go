/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/vvm"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, jdi vvm.IEPJournalIndices, jp vvm.IEPJournalPredicates) {
	provideWLogDatesView(appDefBuilder)
	provideQryJournal(cfg, appDefBuilder, jdi, jp)
	jdi.AddNamed(QNameViewWLogDates.String(), QNameViewWLogDates)
	jdi.AddNamed("", QNameViewWLogDates)                                                      // default index
	jp.AddNamed("all", func(schemas appdef.IAppDef, qName appdef.QName) bool { return true }) // default predicate
}

func provideWLogDatesView(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddView(QNameViewWLogDates).
		AddPartField(field_Year, appdef.DataKind_int32).
		AddClustColumn(field_DayOfYear, appdef.DataKind_int32).
		AddValueField(field_FirstOffset, appdef.DataKind_int64, true).
		AddValueField(field_LastOffset, appdef.DataKind_int64, true)
}

func ProvideWLogDatesAsyncProjectorFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         QNameViewWLogDates,
			Func:         wLogDatesProjector,
			HandleErrors: true,
		}
	}
}
