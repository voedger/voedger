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

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func provideRefIntegrityValidation(sr istructsmem.IStatelessResources) {
	sr.AddProjectors(appdef.SysPackagePath, istructs.Projector{
		Name: qNameRecordsRegistryProjector,
		Func: recordsRegistryProjector,
	})
}

func CheckRefIntegrity(obj istructs.IRowReader, appStructs istructs.IAppStructs, wsid istructs.WSID) (err error) {
	appDef := appStructs.AppDef()
	objQName := obj.AsQName(appdef.SystemField_QName)
	fields := appDef.Type(objQName).(appdef.IWithFields)

	for _, refField := range fields.RefFields() {
		targetID := obj.AsRecordID(refField.Name())
		if targetID == istructs.NullRecordID || targetID.IsRaw() {
			continue
		}
		allowedTargetQNames := appdef.QNamesFrom(refField.Refs()...)
		kb := appStructs.ViewRecords().KeyBuilder(QNameViewRecordsRegistry)
		idHi := CrackID(targetID)
		kb.PutInt64(Field_IDHi, int64(idHi))
		kb.PutRecordID(Field_ID, targetID)
		registryRecord, err := appStructs.ViewRecords().Get(wsid, kb)
		if err == nil {
			if len(allowedTargetQNames) > 0 && !allowedTargetQNames.Contains(registryRecord.AsQName(field_QName)) {
				return wrongQName(targetID, objQName, refField.Name(), registryRecord.AsQName(field_QName), allowedTargetQNames)
			}
			continue
		}
		if !errors.Is(err, istructsmem.ErrRecordNotFound) {
			// notest
			return err
		}
		return fmt.Errorf("%w: record ID %d referenced by %s.%s does not exist", ErrReferentialIntegrityViolation, targetID, objQName, refField.Name())
	}

	return nil
}

func wrongQName(targetID istructs.RecordID, srcQName appdef.QName, srcField string, actualQName appdef.QName, allowedQNames appdef.QNames) error {
	return fmt.Errorf("%w: record ID %d referenced by %s.%s is of QName %s whereas %v QNames are only allowed", ErrReferentialIntegrityViolation,
		targetID, srcQName, srcField, actualQName, allowedQNames)
}

func recordsRegistryProjector(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	appDef := st.AppStructs().AppDef()
	argType := appDef.Type(event.ArgumentObject().QName())
	if argType.Kind() == appdef.TypeKind_ODoc || argType.Kind() == appdef.TypeKind_ORecord {
		if err := writeObjectToRegistry(event.ArgumentObject(), appDef, st, intents, event.WLogOffset()); err != nil {
			// notest
			return err
		}
	}
	for rec := range event.CUDs {
		if !rec.IsNew() {
			continue
		}
		if err := writeObjectToRegistry(rec, appDef, st, intents, event.WLogOffset()); err != nil {
			return err
		}
	}
	return nil
}

func writeObjectToRegistry(root istructs.IRowReader, appDef appdef.IAppDef, st istructs.IState, intents istructs.IIntents, wLogOffsetToStore istructs.Offset) error {
	if err := writeRegistry(st, intents, root.AsRecordID(appdef.SystemField_ID), wLogOffsetToStore, root.AsQName(appdef.SystemField_QName)); err != nil {
		// notest
		return err
	}
	object, ok := root.(istructs.IObject)
	if !ok {
		return nil
	}
	for container := range object.Containers {
		for child := range object.Children(container) {
			elType := appDef.Type(child.QName())
			if elType.Kind() == appdef.TypeKind_ODoc || elType.Kind() == appdef.TypeKind_ORecord {
				if err := writeObjectToRegistry(child, appDef, st, intents, wLogOffsetToStore); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeRegistry(st istructs.IState, intents istructs.IIntents, idToStore istructs.RecordID, wLogOffsetToStore istructs.Offset, qNameToStore appdef.QName) error {
	kb, err := st.KeyBuilder(sys.Storage_View, QNameViewRecordsRegistry)
	if err != nil {
		// notest
		return err
	}
	idHi := CrackID(idToStore)
	kb.PutInt64(Field_IDHi, int64(idHi))
	kb.PutRecordID(Field_ID, idToStore)
	recordsRegistryRecBuilder, err := intents.NewValue(kb)
	if err != nil {
		// notest
		return err
	}
	recordsRegistryRecBuilder.PutInt64(Field_WLogOffset, int64(wLogOffsetToStore))
	recordsRegistryRecBuilder.PutQName(field_QName, qNameToStore)
	return nil
}

func provideRefIntegrityValidator() istructs.CUDValidator {
	return istructs.CUDValidator{
		Match: func(cud istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) bool {
			return cmdQName != QNameCommandInit
		},
		Validate: func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) (err error) {
			if err = CheckRefIntegrity(cudRow, appStructs, wsid); err == nil {
				return nil
			}
			status := http.StatusInternalServerError
			if errors.Is(err, ErrReferentialIntegrityViolation) {
				status = http.StatusBadRequest
			}
			return coreutils.WrapSysError(err, status)
		},
	}
}
