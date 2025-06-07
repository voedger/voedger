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
)

func readWlog(ctx context.Context, wsid istructs.WSID, offset istructs.Offset, count int, appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback,
	appDef appdef.IAppDef) error {
	if !f.acceptAll {
		for field := range f.fields {
			if !wlogDef[field] {
				return fmt.Errorf("field '%s' not found in def", field)
			}
		}
	}
	return appStructs.Events().ReadWLog(ctx, wsid, offset, count, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		data := make(map[string]interface{})

		if f.filter("WlogOffset") {
			data["WlogOffset"] = wlogOffset
		}

		renderDBEvent(data, f, event, appDef, offset)

		bb, err := json.Marshal(data)
		if err != nil {
			return err
		}

		return callback(&result{value: string(bb)})
	})
}
