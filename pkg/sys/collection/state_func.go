/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package collection

import (
	"context"
	"encoding/json"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func provideStateFunc(sr istructsmem.IStatelessResources) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunction(
		qNameQueryState,
		stateFuncExec))
}

func stateFuncExec(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	after := args.ArgumentObject.AsInt64(field_After)

	kb, err := args.State.KeyBuilder(sys.Storage_View, QNameCollectionView)
	if err != nil {
		return err
	}
	kb.PutInt32(Field_PartKey, PartitionKeyCollection)

	data := make(map[string]map[istructs.RecordID]map[string]interface{})
	appDef := args.State.AppStructs().AppDef()
	maxRelevantOffset := int64(0)
	if err = args.State.Read(kb, func(key istructs.IKey, value istructs.IStateValue) (err error) {
		if value.AsInt64(state.ColOffset) <= after {
			return
		}
		record := value.(istructs.IStateViewValue).AsRecord(Field_Record)
		_, ok := data[record.QName().String()]
		if !ok {
			data[record.QName().String()] = make(map[istructs.RecordID]map[string]interface{})
		}
		recordData := coreutils.FieldsToMap(record, appDef, coreutils.Filter(func(name string, kind appdef.DataKind) bool {
			return name != appdef.SystemField_QName && name != appdef.SystemField_Container
		}), coreutils.WithAllFields())
		data[record.QName().String()][record.ID()] = recordData
		if value.AsInt64(state.ColOffset) > maxRelevantOffset {
			maxRelevantOffset = value.AsInt64(state.ColOffset)
		}
		return err
	}); err != nil {
		return
	}
	bb, err := json.Marshal(data)
	if err != nil {
		return
	}
	return callback(&stateObject{data: string(bb), maxRelevantOffset: maxRelevantOffset})
}

type stateObject struct {
	istructs.NullObject
	data              string
	maxRelevantOffset int64
}

func (o stateObject) AsString(string) string { return o.data }

func (o stateObject) AsInt64(string) int64 { return o.maxRelevantOffset }
