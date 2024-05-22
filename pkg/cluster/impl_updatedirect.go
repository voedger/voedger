/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func updateDirect(update update) error {
	if update.qNameTypeKind == appdef.TypeKind_ViewRecord {
		return updateDirect_View(update)
	}
	return updateDirect_Record(update)
}

func updateDirect_Record(update update) error {
	existingRec, err := update.appStructs.Records().Get(update.wsid, true, update.id)
	if err != nil {
		return err
	}
	if existingRec.QName() == appdef.NullQName {
		return fmt.Errorf("record ID %d does not exist", update.id)
	}
	existingFields := coreutils.FieldsToMap(existingRec, update.appStructs.AppDef(), coreutils.WithNonNilsOnly())
	mergedFields := coreutils.MergeMapsMakeFloats64(existingFields, update.setFields)
	return update.appStructs.Records().PutJSON(update.wsid, mergedFields)
}

func updateDirect_View(update update) error {
	kb := update.appStructs.ViewRecords().KeyBuilder(update.qName)
	if err := coreutils.MapToObject(update.key, kb); err != nil {
		return err
	}

	existingViewRec, err := update.appStructs.ViewRecords().Get(update.wsid, kb)
	if err != nil {
		// including "not found" error
		return err
	}

	existingFields := coreutils.FieldsToMap(existingViewRec, update.appStructs.AppDef(), coreutils.WithNonNilsOnly())

	mergedFields := coreutils.MergeMapsMakeFloats64(existingFields, update.setFields, update.key)
	return update.appStructs.ViewRecords().PutJSON(update.wsid, mergedFields)
}

func validateQuery_Direct(update update) error {
	tp := update.appStructs.AppDef().Type(update.qName)
	if containers, ok := tp.(appdef.IContainers); ok {
		if containers.ContainerCount() > 0 {
			// TODO: no design?
			return errors.New("impossible to update a record that has containers")
		}
	}
	typeKindToUpdate := tp.Kind()
	if typeKindToUpdate == appdef.TypeKind_ViewRecord {
		if update.id > 0 {
			return errors.New("record ID must not be provided on view direct update")
		}
		if len(update.key) == 0 {
			return errors.New("full key must be provided on view direct update")
		}
	} else {
		if update.id == 0 {
			return errors.New("record ID must be provided on record direct update")
		}
		if len(update.key) > 0 {
			return errors.New("'where' clause is not allowed on record direct update")
		}
	}
	return nil
}
