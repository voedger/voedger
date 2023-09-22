/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideQryJournal(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Journal"),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "JournalParams")).
			AddField(field_From, appdef.DataKind_int64, true).
			AddField(field_Till, appdef.DataKind_int64, true).
			AddField(Field_EventTypes, appdef.DataKind_string, true).
			AddField(field_IndexForTimestamps, appdef.DataKind_string, false).
			AddField(field_RangeUnit, appdef.DataKind_string, false).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "JournalResult")).
			AddField(Field_Offset, appdef.DataKind_int64, true).
			AddField(Field_EventTime, appdef.DataKind_int64, true).
			AddField(Field_Event, appdef.DataKind_string, true).(appdef.IType).QName(),
		qryJournalExec(ep, appDefBuilder),
	))
}
func qryJournalExec( /*jdi vvm.IEPJournalIndices, jp vvm.IEPJournalPredicates, */ ep extensionpoints.IExtensionPoint, appDef appdef.IAppDef) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		var fo, lo int64
		ji := ep.ExtensionPoint(EPJournalIndices)
		jp := ep.ExtensionPoint(EPJournalPredicates)
		switch args.ArgumentObject.AsString(field_RangeUnit) {
		case rangeUnit_UnixTimestamp:
			fallthrough
		case "":
			fo, lo, err = handleTimestamps(args.ArgumentObject, ji, args.State)
		case rangeUnit_Offset:
			fo, lo, err = handleOffsets(args.ArgumentObject)
		default:
			err = errArgumentTypeNotSupported
		}
		if err != nil {
			return err
		}

		f, err := NewFilter(appDef, strings.Split(args.ArgumentObject.AsString(Field_EventTypes), ","), jp)
		if err != nil {
			return err
		}

		cb := func(_ istructs.IKey, value istructs.IStateValue) (err error) {
			if fo == int64(0) {
				return
			}
			eo, err := NewEventObject(value.AsEvent("").(istructs.IWLogEvent), appDef, f, coreutils.WithNonNilsOnly())
			if err != nil {
				return err
			}
			if !eo.Empty {
				eo.Data[Field_Offset] = value.AsInt64(state.Field_Offset)
				if err := callback(eo); err != nil {
					return err
				}
			}
			return err
		}

		kb, err := args.State.KeyBuilder(state.WLogStorage, appdef.NullQName)
		if err != nil {
			return err
		}
		kb.PutInt64(state.Field_Offset, fo)
		kb.PutInt64(state.Field_Count, lo-fo+1)

		return args.State.Read(kb, cb)
	}
}

func handleTimestamps(args istructs.IObject, epJornalIndices extensionpoints.IExtensionPoint, state istructs.IState) (fo, lo int64, err error) {
	resetTime := func(milli int64) time.Time {
		y, m, d := time.UnixMilli(milli).UTC().Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}

	from := resetTime(args.AsInt64(field_From))
	till := resetTime(args.AsInt64(field_Till))

	idxIntf, ok := epJornalIndices.Find(args.AsString(field_IndexForTimestamps))
	if !ok {
		return 0, 0, errIndexNotSupported
	}
	idx := idxIntf.(appdef.QName)

	return FindOffsetsByTimeRange(from, till, idx, state)
}

// TODO use errors.Join after migration to go 1.20
func handleOffsets(args istructs.IObject) (fo, lo int64, err error) {
	fo = args.AsInt64(field_From)
	lo = args.AsInt64(field_Till)
	if fo <= 0 {
		err = fmt.Errorf("<<from>> %w", errOffsetMustBePositive)
	}
	if lo <= 0 {
		err = fmt.Errorf("<<till>> %w", errOffsetMustBePositive)
	}
	if fo > lo {
		err = errFromOffsetMustBeLowerOrEqualToTillOffset
	}
	return
}
