/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
)

func readPlog(ctx context.Context, wsid istructs.WSID, offset istructs.Offset, count int,
	appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback, appDef appdef.IAppDef, appParts appparts.IAppPartitions) error {
	if !f.acceptAll {
		for field := range f.fields {
			if !plogDef[field] {
				return fmt.Errorf("field '%s' not found in def", field)
			}
		}
	}
	partitionID, err := appParts.AppWorkspacePartitionID(appStructs.AppQName(), wsid)
	if err != nil {
		return err
	}
	return appStructs.Events().ReadPLog(ctx, partitionID, offset, count, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		data := make(map[string]interface{})
		if f.filter("PlogOffset") {
			data["PlogOffset"] = plogOffset
		}
		if f.filter("Workspace") {
			data["Workspace"] = event.Workspace()
		}
		if f.filter("WLogOffset") {
			data["WLogOffset"] = event.WLogOffset()
		}

		renderDBEvent(data, f, event, appDef, event.WLogOffset())

		bb, err := json.Marshal(data)
		if err != nil {
			// notest
			return err
		}

		return callback(&result{value: string(bb)})
	})
}
