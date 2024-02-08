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

func Provide(cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) {
	provideQryJournal(cfg, ep)
	ji := ep.ExtensionPoint(EPJournalIndices)
	ji.AddNamed(QNameViewWLogDates.String(), QNameViewWLogDates)
	ji.AddNamed("", QNameViewWLogDates) // default index
	jp := ep.ExtensionPoint(EPJournalPredicates)
	jp.AddNamed("all", func(schemas appdef.IWorkspace, qName appdef.QName) bool { return true }) // default predicate

	cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorWLogDates,
			Func: wLogDatesProjector,
		}
	})
}
