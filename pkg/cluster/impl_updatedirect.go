/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/dml"
	"github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func updateDirect(update update) error {
	if update.qNameTypeKind == appdef.TypeKind_ViewRecord {
		return updateDirect_View(update)
	}
	return updateDirect_Record(update)
}

func updateDirect_View(update update) (err error) {
	kb := update.appStructs.ViewRecords().KeyBuilder(update.QName)
	if update.Kind == dml.OpKind_DirectInsert {
		kb.PutFromJSON(update.setFields)
	} else if err = coreutils.MapToObject(update.key, kb); err != nil {
		return err
	}

	existingViewRec, err := update.appStructs.ViewRecords().Get(update.wsid, kb)
	if update.Kind == dml.OpKind_DirectInsert {
		if err == nil {
			return coreutils.NewHTTPErrorf(http.StatusConflict, "view record already exists")
		}
		if !errors.Is(err, istructsmem.ErrRecordNotFound) {
			// notest
			return err
		}
	} else if err != nil {
		// including "not found" error
		return err
	}

	existingFields := coreutils.FieldsToMap(existingViewRec, update.appStructs.AppDef(), coreutils.WithNonNilsOnly())

	mergedFields := coreutils.MergeMapsMakeFloats64(existingFields, update.setFields, update.key)
	mergedFields[appdef.SystemField_QName] = update.QName.String() // missing on direct insert
	return update.appStructs.ViewRecords().PutJSON(update.wsid, mergedFields)
}

func updateDirect_Record(update update) error {
	existingRec, err := update.appStructs.Records().Get(update.wsid, true, update.id)
	if err != nil {
		// notest
		return err
	}
	if existingRec.QName() == appdef.NullQName {
		return fmt.Errorf("record ID %d does not exist", update.id)
	}
	existingFields := coreutils.FieldsToMap(existingRec, update.appStructs.AppDef(), coreutils.WithNonNilsOnly())
	mergedFields := coreutils.MergeMapsMakeFloats64(existingFields, update.setFields)
	return update.appStructs.Records().PutJSON(update.wsid, mergedFields)
}

func validateQuery_Direct(update update) error {
	op := "update"
	if update.Kind == dml.OpKind_DirectInsert {
		op = "insert"
	}
	tp := update.appStructs.AppDef().Type(update.QName)
	switch {
	case tp.Kind() == appdef.TypeKind_ViewRecord:
		if update.id > 0 {
			return fmt.Errorf("record ID must not be provided on view direct %s", op)
		}
		if update.Kind == dml.OpKind_DirectInsert {
			if len(update.key) > 0 {
				return errors.New("'where' clause is not allowed on view direct insert")
			}
		} else {
			if len(update.key) == 0 {
				return errors.New("full key must be provided on view direct update")
			}
		}
	case allowedDocsTypeKinds[tp.Kind()]:
		if containers, ok := tp.(appdef.IContainers); ok {
			if containers.ContainerCount() > 0 {
				// TODO: no design?
				return fmt.Errorf("impossible to %s a record that has containers", op)
			}
		}
		if update.Kind == dml.OpKind_DirectInsert {
			return errors.New("direct insert is not allowed for records")
		}
		if update.id == 0 {
			return errors.New("record ID must be provided on record direct update")
		}
		if len(update.key) > 0 {
			return errors.New("'where' clause is not allowed on record direct update")
		}
	default:
		return errors.New("view, CDoc or WDoc only expected")
	}
	return nil
}
