/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func provideQryJournal(sr istructsmem.IStatelessResources, eps map[appdef.AppQName]extensionpoints.IExtensionPoint) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Journal"),
		qryJournalExec(eps),
	))
}
func qryJournalExec(eps map[appdef.AppQName]extensionpoints.IExtensionPoint) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		var fo, lo int64
		ep := eps[args.State.App()]
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

		appDef := args.State.AppStructs().AppDef()
		cb := func(_ istructs.IKey, value istructs.IStateValue) (err error) {
			if fo == int64(0) {
				return
			}
			eo, err := NewEventObject(value.(istructs.IStateWLogValue).AsEvent(), appDef, f, coreutils.WithNonNilsOnly())
			if err != nil {
				return err
			}
			if !eo.Empty {
				eo.Data[Field_Offset] = value.AsInt64(sys.Storage_WLog_Field_Offset)
				if err := callback(eo); err != nil {
					return err
				}
			}
			return err
		}

		kb, err := args.State.KeyBuilder(sys.Storage_WLog, appdef.NullQName)
		if err != nil {
			return err
		}
		kb.PutInt64(sys.Storage_WLog_Field_Offset, fo)
		kb.PutInt64(sys.Storage_WLog_Field_Count, lo-fo+1)

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
func handleOffsets(args istructs.IObject) (fo, lo int64, err error) {
	fo = args.AsInt64(field_From)
	lo = args.AsInt64(field_Till)
	errs := make([]error, 0)
	if fo <= 0 {
		errs = append(errs, fmt.Errorf("'from' %w", errOffsetMustBePositive))
	}
	if lo <= 0 {
		errs = append(errs, fmt.Errorf("'till' %w", errOffsetMustBePositive))
	}
	if fo > lo {
		errs = append(errs, errFromOffsetMustBeLowerOrEqualToTillOffset)
	}
	err = errors.Join(errs...)
	return
}
