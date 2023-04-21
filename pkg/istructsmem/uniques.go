/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

type implIUnique struct {
	fields []string
	qName  schemas.QName
}

func newUniques() *implIUniques {
	return &implIUniques{uniques: map[schemas.QName][]istructs.IUnique{}}
}

type implIUniques struct {
	uniques map[schemas.QName][]istructs.IUnique
}

func (u *implIUnique) Fields() []string {
	return u.fields
}

func (u *implIUnique) QName() schemas.QName {
	return u.qName
}

func (u *implIUniques) Add(name schemas.QName, fieldNames []string) {
	u.uniques[name] = append(u.uniques[name], &implIUnique{fields: fieldNames, qName: name})
}

func (u *implIUniques) GetAll(name schemas.QName) (uniques []istructs.IUnique) {
	return u.uniques[name]
}

// returns an Unique that euqals to provided keyFieldsSet ignoring order
// nil means not found
// panics if a duplicate key field name is met in keyFieldsSet
func (u implIUniques) GetForKeySet(qName schemas.QName, keyFieldsSet []string) istructs.IUnique {
	for _, unique := range u.uniques[qName] {
		if len(unique.Fields()) != len(keyFieldsSet) {
			continue
		}
		m := map[string]bool{}
		for _, f := range keyFieldsSet {
			if _, ok := m[f]; ok {
				panic("duplicate field " + f)
			}
			m[f] = false
			for _, uf := range unique.Fields() {
				if uf == f {
					m[f] = true
				}
			}
		}
		match := true
		for _, val := range m {
			if !val {
				match = false
				break
			}
		}
		if match {
			return unique
		}
	}
	return nil
}

type fieldDesc struct {
	kind       schemas.DataKind
	isRequired bool
}

func (u implIUniques) validate(cfg *AppConfigType) error {
	for qName, uniques := range u.uniques {
		s := cfg.Schemas.SchemaByName(qName)
		if s == nil {
			return uniqueError(qName, ErrUnknownSchemaQName, "")
		}
		switch s.Kind() {
		case schemas.SchemaKind_ViewRecord, schemas.SchemaKind_ViewRecord_PartitionKey, schemas.SchemaKind_ViewRecord_ClusteringColumns,
			schemas.SchemaKind_ViewRecord_Value, schemas.SchemaKind_Object, schemas.SchemaKind_Element,
			schemas.SchemaKind_QueryFunction, schemas.SchemaKind_CommandFunction:
			return uniqueError(qName, ErrSchemaKindMayNotHaveUniques, "")
		}
		sf := map[string]fieldDesc{}
		s.EnumFields(func(fld schemas.Field) {
			sf[fld.Name()] = fieldDesc{
				kind:       fld.DataKind(),
				isRequired: fld.Required(),
			}
		})
		duplicateUnique := []map[string]bool{}
		for _, unique := range uniques {
			if len(unique.Fields()) == 0 {
				return uniqueError(qName, ErrEmptySetOfKeyFields, "")
			}
			duplicateField := map[string]bool{}
			varSizeFieldsAmount := 0
			for _, f := range unique.Fields() {
				if duplicateField[f] {
					return uniqueError(qName, ErrKeyFieldIsUsedMoreThanOnce, f)
				}
				fieldDesc, ok := sf[f]
				if !ok {
					return uniqueError(qName, ErrUnknownKeyField, f)
				}
				if fieldDesc.kind == schemas.DataKind_string || fieldDesc.kind == schemas.DataKind_bytes {
					varSizeFieldsAmount++
				}
				if varSizeFieldsAmount > 1 {
					return uniqueError(qName, ErrKeyMustHaveNotMoreThanOneVarSizeField, f)
				}
				if !fieldDesc.isRequired {
					return uniqueError(qName, ErrKeyFieldMustBeRequired, f)
				}
				duplicateField[f] = true
			}
			duplicateUnique = append(duplicateUnique, duplicateField)
		}
		for i, duLeft := range duplicateUnique {
			for j, duRight := range duplicateUnique {
				if j == i || len(duRight) != len(duLeft) {
					continue
				}
				hasDiffs := false
				for fLeft := range duLeft {
					if !duRight[fLeft] {
						hasDiffs = true
						break
					}
				}
				if !hasDiffs {
					return uniqueError(qName, ErrUniquesHaveSameFields, "")
				}
			}
		}
	}
	return nil
}

func uniqueError(qName schemas.QName, err error, name string) error {
	mes := "unique on %s: %w"
	if len(name) > 0 {
		mes += ": %s"
	}
	return fmt.Errorf(mes, qName, err, name)
}
