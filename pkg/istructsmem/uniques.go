/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type implIUnique struct {
	fields []string
	qName  appdef.QName
}

func newUniques() *implIUniques {
	return &implIUniques{uniques: map[appdef.QName][]istructs.IUnique{}}
}

type implIUniques struct {
	uniques map[appdef.QName][]istructs.IUnique
}

func (u *implIUnique) Fields() []string {
	return u.fields
}

func (u *implIUnique) QName() appdef.QName {
	return u.qName
}

func (u *implIUniques) Add(name appdef.QName, fieldNames []string) {
	u.uniques[name] = append(u.uniques[name], &implIUnique{fields: fieldNames, qName: name})
}

func (u *implIUniques) GetAll(name appdef.QName) (uniques []istructs.IUnique) {
	return u.uniques[name]
}

type fieldDesc struct {
	kind       appdef.DataKind
	isRequired bool
}

func (u implIUniques) validate(cfg *AppConfigType) error {

	uniqueError := func(qName appdef.QName, wrap error, msg string, args ...any) error {
		err := fmt.Sprintf(msg, args...)
		return fmt.Errorf("%v unique error: %s: %w", qName, err, wrap)
	}

	for qName, uniques := range u.uniques {
		d := cfg.AppDef.DefByName(qName)
		if d == nil {
			return uniqueError(qName, appdef.ErrNameNotFound, "definition «%v» not found", qName)
		}
		if !d.Kind().UniquesAvailable() {
			return uniqueError(qName, appdef.ErrInvalidDefKind, "definition kind «%v» unable uniques", d.Kind())
		}

		if len(uniques) > 1 {
			return uniqueError(qName, appdef.ErrTooManyUniques, "only one unique per definition available")
		}

		unique := uniques[0]
		if len(unique.Fields()) > 1 {
			return uniqueError(qName, appdef.ErrTooManyFields, "only one field per unique available")
		}

		name := unique.Fields()[0]
		fld := d.Field(name)

		if fld == nil {
			return uniqueError(qName, appdef.ErrNameNotFound, "field «%s» not found", name)
		}

		if !fld.Required() {
			return uniqueError(qName, appdef.ErrWrongDefStruct, "field «%s» must be required", name)
		}
	}
	return nil
}
