/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	provideWLogDatesView(appDefBuilder)
	provideQryJournal(cfg, appDefBuilder, ep)
	ji := ep.ExtensionPoint(EPJournalIndices)
	ji.AddNamed(QNameViewWLogDates.String(), QNameViewWLogDates)
	ji.AddNamed("", QNameViewWLogDates) // default index
	jp := ep.ExtensionPoint(EPJournalPredicates)
	jp.AddNamed("all", func(schemas appdef.IAppDef, qName appdef.QName) bool { return true }) // default predicate

	appDefBuilder.AddObject(QNameProjectorWLogDates)

	cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         QNameProjectorWLogDates,
			Func:         wLogDatesProjector,
			HandleErrors: true,
		}
	})
}

func provideWLogDatesView(app appdef.IAppDefBuilder) {
	view := app.AddView(QNameViewWLogDates)
	view.Key().Partition().AddField(field_Year, appdef.DataKind_int32)
	view.Key().ClustCols().AddField(field_DayOfYear, appdef.DataKind_int32)
	view.Value().
		AddField(field_FirstOffset, appdef.DataKind_int64, true).
		AddField(field_LastOffset, appdef.DataKind_int64, true)
}
