/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideRefIntegrityValidation(cfg *istructsmem.AppConfigType) {
	cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: appdef.NewQName(appdef.SysPackage, "ORecordsRegistryProjector"),
			Func: provideRecordsRegistryProjector(cfg),
		}
	})
	// cfg.AddCUDValidators(provideRefIntegrityValidator())
	cfg.AddEventValidators(refIntegrityValidator)
}

func refIntegrityValidator(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
	if coreutils.IsDummyWS(wsid) || rawEvent.QName() == QNameCommandInit {
		return nil
	}
	argType := appStructs.AppDef().Type(rawEvent.ArgumentObject().QName())
	// if argType.Kind() == appdef.TypeKind_ODoc || argType.Kind() == ORecord
}

func provideRecordsRegistryProjector(cfg *istructsmem.AppConfigType) func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		argType := cfg.AppDef.Type(event.ArgumentObject().QName())
		if argType.Kind() != appdef.TypeKind_ODoc && argType.Kind() != appdef.TypeKind_ORecord {
			return nil
		}

		return event.CUDs(func(rec istructs.ICUDRow) error {
			if !rec.IsNew() {
				return nil
			}
			recType := cfg.AppDef.Type(rec.QName())
			if recType.Kind() != appdef.TypeKind_ODoc && recType.Kind() != appdef.TypeKind_ORecord {
				return nil
			}
			kb, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewORecordsRegistry)
			if err != nil {
				// notest
				return err
			}
			kb.PutRecordID(field_ID, rec.ID())
			recordsRegistryRecBuilder, err := intents.NewValue(kb)
			if err != nil {
				// notest
				return err
			}
			recordsRegistryRecBuilder.PutInt32(field_Dummy, 1)
			recordsRegistryRecBuilder.PutInt64(field_WLogOffset, int64(event.WLogOffset()))
			return nil
		})
	}
}

func provideRefIntegrityValidator() istructs.CUDValidator {
	return istructs.CUDValidator{
		MatchFunc: func(qName appdef.QName, wsid istructs.WSID, cmdQName appdef.QName) bool {
			return !coreutils.IsDummyWS(wsid) && cmdQName != QNameCommandInit
		},
		Validate: func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) (err error) {
			if err = CheckRefIntegrity(cudRow, appStructs, wsid); err == nil {
				return nil
			}
			if errors.Is(err, ErrReferentialIntegrityViolation) {
				return coreutils.WrapSysError(err, http.StatusBadRequest)
			}
			// notest
			return coreutils.WrapSysError(err, http.StatusInternalServerError)
		},
	}
}

func CheckRefIntegrity(obj istructs.IRowReader, appStructs istructs.IAppStructs, wsid istructs.WSID) (err error) {
	appDef := appStructs.AppDef()
	qName := obj.AsQName(appdef.SystemField_QName)
	t := appDef.Type(qName)
	fields, ok := t.(appdef.IFields)
	if !ok {
		return nil
	}
	return iterate.ForEachError(fields.RefFields, func(refField appdef.IRefField) error {
		actualRefID := obj.AsRecordID(refField.Name())
		if actualRefID == istructs.NullRecordID || actualRefID.IsRaw() {
			return nil
		}
		allowedTargetQNames := refField.Refs()

		refToRPossible := len(allowedTargetQNames) == 0
		refToOPossible := len(allowedTargetQNames) == 0
		for _, allowedTargetQName := range allowedTargetQNames {
			targetType := appDef.Type(allowedTargetQName)
			switch targetType.Kind() {
			case appdef.TypeKind_ODoc, appdef.TypeKind_ORecord:
				refToOPossible = true
			default:
				refToRPossible = true
			}
		}
		if refToRPossible {
			actualRefRec, err := appStructs.Records().Get(wsid, true, actualRefID)
			if err != nil {
				// notest
				return err
			}
			if actualRefRec.QName() != appdef.NullQName {
				if len(allowedTargetQNames) > 0 && !slices.Contains(allowedTargetQNames, actualRefRec.QName()) {
					return fmt.Errorf("%w: record ID %d referenced by %s.%s is of QName %s whereas %v QNames are only allowed", ErrReferentialIntegrityViolation,
						actualRefID, qName, refField.Name(), actualRefRec.QName(), refField.Refs())
				}
				return nil
			}
		}
		if refToOPossible {
			kb := appStructs.ViewRecords().KeyBuilder(QNameViewORecordsRegistry)
			kb.PutRecordID(field_ID, actualRefID)
			kb.PutInt32(field_Dummy, 1)
			_, err := appStructs.ViewRecords().Get(wsid, kb)
			if err == nil {
				return nil
			}
			if !errors.Is(err, istructsmem.ErrRecordNotFound) {
				// notest
				return err
			}
		}
		return fmt.Errorf("%w: record ID %d referenced by %s.%s does not exist", ErrReferentialIntegrityViolation, actualRefID, qName, refField.Name())
	})
}

// func CheckRefIntegrity(obj istructs.IRowReader, appStructs istructs.IAppStructs, wsid istructs.WSID) (err error) {
// 	appDef := appStructs.AppDef()
// 	qName := obj.AsQName(appdef.SystemField_QName)
// 	t := appDef.Type(qName)
// 	fields, ok := t.(appdef.IFields)
// 	if !ok {
// 		return nil
// 	}
// 	return iterate.ForEachError(fields.RefFields, func(refField appdef.IRefField) error {
// 		actualRefID := obj.AsRecordID(refField.Name())
// 		if actualRefID == istructs.NullRecordID || actualRefID.IsRaw() {
// 			return nil
// 		}
// 		allowedTargetQNames := refField.Refs()

// 		refToRPossible := len(allowedTargetQNames) == 0
// 		refToOPossible := len(allowedTargetQNames) == 0
// 		for _, allowedTargetQName := range allowedTargetQNames {
// 			targetType := appDef.Type(allowedTargetQName)
// 			switch targetType.Kind() {
// 			case appdef.TypeKind_ODoc, appdef.TypeKind_ORecord:
// 				refToOPossible = true
// 			default:
// 				refToRPossible = true
// 			}
// 		}
// 		if refToRPossible {
// 			actualRefRec, err := appStructs.Records().Get(wsid, true, actualRefID)
// 			if err != nil {
// 				// notest
// 				return err
// 			}
// 			if actualRefRec.QName() != appdef.NullQName {
// 				if len(allowedTargetQNames) > 0 && !slices.Contains(allowedTargetQNames, actualRefRec.QName()) {
// 					return fmt.Errorf("%w: record ID %d referenced by %s.%s is of QName %s whereas %v QNames are only allowed", ErrReferentialIntegrityViolation,
// 						actualRefID, qName, refField.Name(), actualRefRec.QName(), refField.Refs())
// 				}
// 				return nil
// 			}
// 		}
// 		if refToOPossible {
// 			kb := appStructs.ViewRecords().KeyBuilder(QNameViewORecordsRegistry)
// 			kb.PutRecordID(field_ID, actualRefID)
// 			kb.PutInt32(field_Dummy, 1)
// 			_, err := appStructs.ViewRecords().Get(wsid, kb)
// 			if err == nil {
// 				return nil
// 			}
// 			if !errors.Is(err, istructsmem.ErrRecordNotFound) {
// 				// notest
// 				return err
// 			}
// 		}
// 		return fmt.Errorf("%w: record ID %d referenced by %s.%s does not exist", ErrReferentialIntegrityViolation, actualRefID, qName, refField.Name())
// 	})
// }
