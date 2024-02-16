/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func readWlog(ctx context.Context, wsid istructs.WSID, offset istructs.Offset, count int, appStructs istructs.IAppStructs, f *filter, callback istructs.ExecQueryCallback) error {
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
		if f.filter("QName") {
			data["QName"] = event.QName().String()
		}
		if f.filter("ArgumentObject") {
			data["ArgumentObject"] = coreutils.ObjectToMap(event.ArgumentObject(), appStructs.AppDef())
		}
		if f.filter("CUDs") {
			cuds := make([]map[string]interface{}, 0)
			event.CUDs(func(rec istructs.ICUDRow) {
				cudData := make(map[string]interface{})
				cudData["sys.ID"] = rec.ID()
				cudData["sys.QName"] = rec.QName().String()
				cudData["IsNew"] = rec.IsNew()
				cudData["fields"] = coreutils.FieldsToMap(rec, appStructs.AppDef())
				cuds = append(cuds, cudData)
			})
			data["CUDs"] = cuds
		}
		if f.filter("RegisteredAt") {
			data["RegisteredAt"] = event.RegisteredAt()
		}
		if f.filter("Synced") {
			data["Synced"] = event.Synced()
		}
		if f.filter("DeviceID") {
			data["DeviceID"] = event.DeviceID()
		}
		if f.filter("SyncedAt") {
			data["SyncedAt"] = event.SyncedAt()
		}
		if f.filter("Error") {
			if event.Error() != nil {
				errorData := make(map[string]interface{})
				errorData["ErrStr"] = event.Error().ErrStr()
				errorData["QNameFromParams"] = event.Error().QNameFromParams().String()
				errorData["ValidEvent"] = event.Error().ValidEvent()
				errorData["OriginalEventBytes"] = event.Error().OriginalEventBytes()
				data["Error"] = errorData
			}
		}

		bb, err := json.Marshal(data)
		if err != nil {
			return err
		}

		return callback(&result{value: string(bb)})
	})
}
