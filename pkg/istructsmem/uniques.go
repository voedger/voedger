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
	qName  istructs.QName
}

func newUniques() *implIUniques {
	return &implIUniques{uniques: map[istructs.QName][]istructs.IUnique{}}
}

type implIUniques struct {
	uniques map[istructs.QName][]istructs.IUnique
}

func (u *implIUnique) Fields() []string {
	return u.fields
}

func (u *implIUnique) QName() istructs.QName {
	return u.qName
}

func (u *implIUniques) Add(name istructs.QName, fieldNames []string) {
	u.uniques[name] = append(u.uniques[name], &implIUnique{fields: fieldNames, qName: name})
}

func (u *implIUniques) GetAll(name istructs.QName) (uniques []istructs.IUnique) {
	return u.uniques[name]
}

// returns an Unique that euqals to provided keyFieldsSet ignoring order
// nil means not found
// panics if a duplicate key field name is met in keyFieldsSet
func (u implIUniques) GetForKeySet(qName istructs.QName, keyFieldsSet []string) istructs.IUnique {
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
	kind       istructs.DataKindType
	isRequired bool
}

func (u implIUniques) validate(cfg *AppConfigType) error {
	for qName, uniques := range u.uniques {
		s := cfg.Schemas.Schema(qName)
		if s == schemas.NullSchema {
			return uniqueError(qName, ErrUnknownSchemaQName, "")
		}
		switch s.Kind() {
		case istructs.SchemaKind_ViewRecord, istructs.SchemaKind_ViewRecord_PartitionKey, istructs.SchemaKind_ViewRecord_ClusteringColumns,
			istructs.SchemaKind_ViewRecord_Value, istructs.SchemaKind_Object, istructs.SchemaKind_Element,
			istructs.SchemaKind_QueryFunction, istructs.SchemaKind_CommandFunction:
			return uniqueError(qName, ErrSchemaKindMayNotHaveUniques, "")
		}
		sf := map[string]fieldDesc{}
		s.ForEachField(func(field istructs.IFieldDescr) {
			sf[field.Name()] = fieldDesc{
				kind:       field.DataKind(),
				isRequired: field.Required(),
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
				if fieldDesc.kind == istructs.DataKind_string || fieldDesc.kind == istructs.DataKind_bytes {
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

func uniqueError(qName istructs.QName, err error, name string) error {
	mes := "unique on %s: %w"
	if len(name) > 0 {
		mes += ": %s"
	}
	return fmt.Errorf(mes, qName, err, name)
}
