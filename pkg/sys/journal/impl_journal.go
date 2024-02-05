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

func provideQryJournal(cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Journal"),
		qryJournalExec(ep),
	))
}
func qryJournalExec(ep extensionpoints.IExtensionPoint) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
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

		f, err := NewFilter(args.Workspace, strings.Split(args.ArgumentObject.AsString(Field_EventTypes), ","), jp)
		if err != nil {
			return err
		}

		cb := func(_ istructs.IKey, value istructs.IStateValue) (err error) {
			if fo == int64(0) {
				return
			}
			eo, err := NewEventObject(value.AsEvent("").(istructs.IWLogEvent), args.Workspace, f, coreutils.WithNonNilsOnly())
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

		kb, err := args.State.KeyBuilder(state.WLog, appdef.NullQName)
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
