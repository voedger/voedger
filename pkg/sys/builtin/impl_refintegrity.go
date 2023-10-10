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
			Name: appdef.NewQName(appdef.SysPackage, "RecordsRegistryProjector"),
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
		kb := appStructs.ViewRecords().KeyBuilder(QNameViewRecordsRegistry)
		idHi, _ := istructsmem.CrackRecordID(targetID)
		kb.PutInt64(field_IDHi, int64(idHi))
		kb.PutRecordID(field_ID, targetID)
		registryRecord, err := appStructs.ViewRecords().Get(wsid, kb)
		if err == nil {
			if len(allowedTargetQNames) > 0 && !slices.Contains(allowedTargetQNames, registryRecord.AsQName(field_QName)) {
				return wrongQName(targetID, objQName, refField.Name(), registryRecord.AsQName(field_QName), refField.Refs())
			}
			return nil
		}
		if !errors.Is(err, istructsmem.ErrRecordNotFound) {
			// notest
			return err
		}
		return fmt.Errorf("%w: record ID %d referenced by %s.%s does not exist", ErrReferentialIntegrityViolation, targetID, objQName, refField.Name())
	})
}

func wrongQName(targetID istructs.RecordID, srcQName appdef.QName, srcField string, actualQName appdef.QName, allowedQNames []appdef.QName) error {
	return fmt.Errorf("%w: record ID %d referenced by %s.%s is of QName %s whereas %v QNames are only allowed", ErrReferentialIntegrityViolation,
		targetID, srcQName, srcField, actualQName, allowedQNames)
}

func provideRecordsRegistryProjector(cfg *istructsmem.AppConfigType) func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		argType := cfg.AppDef.Type(event.ArgumentObject().QName())
		if argType.Kind() == appdef.TypeKind_ODoc || argType.Kind() == appdef.TypeKind_ORecord {
			if err := writeObjectToORegistry(event.ArgumentObject(), cfg.AppDef, st, intents, event.WLogOffset()); err != nil {
				// notest
				return err
			}
		}
		return event.CUDs(func(rec istructs.ICUDRow) error {
			if !rec.IsNew() {
				return nil
			}
			return writeObjectToORegistry(rec, cfg.AppDef, st, intents, event.WLogOffset())
		})
	}
}

func writeObjectToORegistry(root istructs.IRowReader, appDef appdef.IAppDef, st istructs.IState, intents istructs.IIntents, wLogOffsetToStore istructs.Offset) error {
	if err := writeRegistry(st, intents, root.AsRecordID(appdef.SystemField_ID), wLogOffsetToStore, root.AsQName(appdef.SystemField_QName)); err != nil {
		// notest
		return err
	}
	element, ok := root.(istructs.IElement)
	if !ok {
		return nil
	}
	return iterate.ForEachError(element.Containers, func(container string) (err error) {
		element.Elements(container, func(el istructs.IElement) {
			if err != nil {
				// notest
				return
			}
			elType := appDef.Type(el.QName())
			if elType.Kind() != appdef.TypeKind_ODoc && elType.Kind() != appdef.TypeKind_ORecord {
				return
			}
			err = writeObjectToORegistry(el, appDef, st, intents, wLogOffsetToStore)
		})
		return err
	})
}

func writeRegistry(st istructs.IState, intents istructs.IIntents, idToStore istructs.RecordID, wLogOffsetToStore istructs.Offset, qNameToStore appdef.QName) error {
	kb, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewRecordsRegistry)
	if err != nil {
		// notest
		return err
	}
	idHi, _ := istructsmem.CrackRecordID(idToStore)
	kb.PutInt64(field_IDHi, int64(idHi))
	kb.PutRecordID(field_ID, idToStore)
	recordsRegistryRecBuilder, err := intents.NewValue(kb)
	if err != nil {
		// notest
		return err
	}
	recordsRegistryRecBuilder.PutInt64(field_WLogOffset, int64(wLogOffsetToStore))
	recordsRegistryRecBuilder.PutQName(field_QName, qNameToStore)
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
