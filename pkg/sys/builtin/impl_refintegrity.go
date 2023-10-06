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
	cfg.AddCUDValidators(provideRefIntegrityValidator())
}

func CheckRefIntegrity(obj istructs.IRowReader, appStructs istructs.IAppStructs, wsid istructs.WSID) (err error) {
	appDef := appStructs.AppDef()
	objQName := obj.AsQName(appdef.SystemField_QName)
	fields := appDef.Type(objQName).(appdef.IFields)
	return iterate.ForEachError(fields.RefFields, func(refField appdef.IRefField) error {
		targetID := obj.AsRecordID(refField.Name())
		if targetID == istructs.NullRecordID || targetID.IsRaw() {
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
			targetRec, err := appStructs.Records().Get(wsid, true, targetID)
			if err != nil {
				// notest
				return err
			}
			if targetRec.QName() != appdef.NullQName {
				if len(allowedTargetQNames) > 0 && !slices.Contains(allowedTargetQNames, targetRec.QName()) {
					return fmt.Errorf("%w: record ID %d referenced by %s.%s is of QName %s whereas %v QNames are only allowed", ErrReferentialIntegrityViolation,
						targetID, objQName, refField.Name(), targetRec.QName(), refField.Refs())
				}
				return nil
			}
		}
		if refToOPossible {
			kb := appStructs.ViewRecords().KeyBuilder(QNameViewORecordsRegistry)
			kb.PutRecordID(field_ID, targetID)
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
		return fmt.Errorf("%w: record ID %d referenced by %s.%s does not exist", ErrReferentialIntegrityViolation, targetID, objQName, refField.Name())
	})
}

func provideRecordsRegistryProjector(cfg *istructsmem.AppConfigType) func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		argType := cfg.AppDef.Type(event.ArgumentObject().QName())
		if argType.Kind() != appdef.TypeKind_ODoc && argType.Kind() != appdef.TypeKind_ORecord {
			return nil
		}
		return writeObjectToORegistry(event.ArgumentObject(), cfg.AppDef, st, intents, event.WLogOffset())
	}
}

func writeObjectToORegistry(root istructs.IElement, appDef appdef.IAppDef, st istructs.IState, intents istructs.IIntents, wLogOffsetToStore istructs.Offset) error {
	if err := writeORegistry(st, intents, root.AsRecordID(appdef.SystemField_ID), wLogOffsetToStore); err != nil {
		// notest
		return err
	}
	return iterate.ForEachError(root.Containers, func(container string) (err error) {
		root.Elements(container, func(el istructs.IElement) {
			if err != nil {
				// notest
				return
			}
			elType := appDef.Type(el.QName())
			if elType.Kind() != appdef.TypeKind_ODoc && elType.Kind() != appdef.TypeKind_ORecord {
				return
			}
			err = writeORegistry(st, intents, el.AsRecordID(appdef.SystemField_ID), wLogOffsetToStore)
		})
		return err
	})
}

func writeORegistry(st istructs.IState, intents istructs.IIntents, idToStore istructs.RecordID, wLogOffsetToStore istructs.Offset) error {
	kb, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewORecordsRegistry)
	if err != nil {
		// notest
		return err
	}
	kb.PutRecordID(field_ID, idToStore)
	kb.PutInt32(field_Dummy, 1)
	recordsRegistryRecBuilder, err := intents.NewValue(kb)
	if err != nil {
		// notest
		return err
	}
	recordsRegistryRecBuilder.PutInt64(field_WLogOffset, int64(wLogOffsetToStore))
	return nil
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
