/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func readPlog(ctx context.Context, wsid istructs.WSID, numCommandProcessors coreutils.CommandProcessorsCount, offset istructs.Offset, count int, appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback,
	iws appdef.IWorkspace) error {
	if !f.acceptAll {
		for field := range f.fields {
			if !plogDef[field] {
				return fmt.Errorf("field '%s' not found in def", field)
			}
		}
	}
	return appStructs.Events().ReadPLog(ctx, coreutils.PartitionID(wsid, numCommandProcessors), offset, count, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
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

		renderDbEvent(data, f, event, iws)

		bb, err := json.Marshal(data)
		if err != nil {
			return err
		}

		return callback(&result{value: string(bb)})
	})
}
