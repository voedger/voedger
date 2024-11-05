/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/set"
)

// Returns iterator over types by kind.
//
// Types are visited in alphabetic order.
func TypesByKind(types ITypes, kind TypeKind) func(func(IType) bool) {
	return func(visit func(IType) bool) {
		for t := range types.Types {
			if t.Kind() == kind {
				if !visit(t) {
					break
				}
			}
		}
	}
}

// Returns iterator over types by kinds set.
//
// Types are visited in alphabetic order.
func TypesByKinds(types ITypes, kinds TypeKindSet) func(func(IType) bool) {
	return func(visit func(IType) bool) {
		for t := range types.Types {
			if kinds.Contains(t.Kind()) {
				if !visit(t) {
					break
				}
			}
		}
	}
}

// Returns type by name.
//
// Returns nil if type not found.
func TypeByName(types IFindType, name QName) IType {
	if t := types.Type(name); t != NullType {
		return t
	}
	return nil
}

// Returns type by name and kind.
//
// Returns nil if type not found.
func TypeByNameAndKind(types IFindType, name QName, kind TypeKind) IType {
	if t := types.Type(name); t.Kind() == kind {
		return t
	}
	return nil
}

// Returns CDoc by name.
//
// Returns nil if CDoc not found.
func CDoc(types IFindType, name QName) ICDoc {
	if t := TypeByNameAndKind(types, name, TypeKind_CDoc); t != nil {
		return t.(ICDoc)
	}
	return nil
}

// Returns iterator over CDocs.
//
// CDocs are visited in alphabetic order.
func CDocs(types ITypes) func(func(ICDoc) bool) {
	return func(visit func(ICDoc) bool) {
		for t := range TypesByKind(types, TypeKind_CDoc) {
			if !visit(t.(ICDoc)) {
				break
			}
		}
	}
}

// Returns Command by name.
//
// Returns nil if Command not found.
func Command(types IFindType, name QName) ICommand {
	if t := TypeByNameAndKind(types, name, TypeKind_Command); t != nil {
		return t.(ICommand)
	}
	return nil
}

// Returns iterator over Commands.
//
// Command are visited in alphabetic order.
func Commands(types ITypes) func(func(ICommand) bool) {
	return func(visit func(ICommand) bool) {
		for t := range TypesByKind(types, TypeKind_Command) {
			if !visit(t.(ICommand)) {
				break
			}
		}
	}
}

// Returns CRecord by name.
//
// Returns nil if CRecord not found.
func CRecord(types IFindType, name QName) ICRecord {
	if t := TypeByNameAndKind(types, name, TypeKind_CRecord); t != nil {
		return t.(ICRecord)
	}
	return nil
}

// Returns iterator over CRecords.
//
// CRecords are visited in alphabetic order.
func CRecords(types ITypes) func(func(ICRecord) bool) {
	return func(visit func(ICRecord) bool) {
		for t := range TypesByKind(types, TypeKind_CRecord) {
			if !visit(t.(ICRecord)) {
				break
			}
		}
	}
}

// Returns Data type by name.
//
// Returns nil if Data not found.
func Data(types IFindType, name QName) IData {
	if t := TypeByNameAndKind(types, name, TypeKind_Data); t != nil {
		return t.(IData)
	}
	return nil
}

// Returns iterator over Data types.
//
// Data types are visited in alphabetic order.
func DataTypes(types ITypes) func(func(IData) bool) {
	return func(visit func(IData) bool) {
		for t := range TypesByKind(types, TypeKind_Data) {
			if !visit(t.(IData)) {
				break
			}
		}
	}
}

// Returns GDoc by name.
//
// Returns nil if GDoc not found.
func GDoc(types IFindType, name QName) IGDoc {
	if t := TypeByNameAndKind(types, name, TypeKind_GDoc); t != nil {
		return t.(IGDoc)
	}
	return nil
}

// Returns iterator over GDocs.
//
// GDocs are visited in alphabetic order.
func GDocs(types ITypes) func(func(IGDoc) bool) {
	return func(visit func(IGDoc) bool) {
		for t := range TypesByKind(types, TypeKind_GDoc) {
			if !visit(t.(IGDoc)) {
				break
			}
		}
	}
}

// Returns GRecord by name.
//
// Returns nil if GRecord not found.
func GRecord(types IFindType, name QName) IGRecord {
	if t := TypeByNameAndKind(types, name, TypeKind_GRecord); t != nil {
		return t.(IGRecord)
	}
	return nil
}

// Returns iterator over GRecords.
//
// GRecords are visited in alphabetic order.
func GRecords(types ITypes) func(func(IGRecord) bool) {
	return func(visit func(IGRecord) bool) {
		for t := range TypesByKind(types, TypeKind_GRecord) {
			if !visit(t.(IGRecord)) {
				break
			}
		}
	}
}

// Returns Object by name.
//
// Returns nil if Object not found.
func Object(types IFindType, name QName) IObject {
	if t := TypeByNameAndKind(types, name, TypeKind_Object); t != nil {
		return t.(IObject)
	}
	return nil
}

// Returns iterator over Objects.
//
// Objects are visited in alphabetic order.
func Objects(types ITypes) func(func(IObject) bool) {
	return func(visit func(IObject) bool) {
		for t := range TypesByKind(types, TypeKind_Object) {
			if !visit(t.(IObject)) {
				break
			}
		}
	}
}

// Returns ODoc by name.
//
// Returns nil if ODoc not found.
func ODoc(types IFindType, name QName) IODoc {
	if t := TypeByNameAndKind(types, name, TypeKind_ODoc); t != nil {
		return t.(IODoc)
	}
	return nil
}

// Returns iterator over ODocs.
//
// ODocs are visited in alphabetic order.
func ODocs(types ITypes) func(func(IODoc) bool) {
	return func(visit func(IODoc) bool) {
		for t := range TypesByKind(types, TypeKind_ODoc) {
			if !visit(t.(IODoc)) {
				break
			}
		}
	}
}

// Returns ORecord by name.
//
// Returns nil if ORecord not found.
func ORecord(types IFindType, name QName) IORecord {
	if t := TypeByNameAndKind(types, name, TypeKind_ORecord); t != nil {
		return t.(IORecord)
	}
	return nil
}

// Returns iterator over ORecords.
//
// ORecords are visited in alphabetic order.
func ORecords(types ITypes) func(func(IORecord) bool) {
	return func(visit func(IORecord) bool) {
		for t := range TypesByKind(types, TypeKind_ORecord) {
			if !visit(t.(IORecord)) {
				break
			}
		}
	}
}

// Returns system Data type (sys.int32, sys.float654, etc.) by data kind.
//
// Returns nil if not found.
func SysData(types IFindType, k DataKind) IData {
	if t := TypeByNameAndKind(types, SysDataName(k), TypeKind_Data); t != nil {
		return t.(IData)
	}
	return nil
}

// Returns WDoc by name.
//
// Returns nil if WDoc not found.
func WDoc(types IFindType, name QName) IWDoc {
	if t := TypeByNameAndKind(types, name, TypeKind_WDoc); t != nil {
		return t.(IWDoc)
	}
	return nil
}

// Returns iterator over WDocs.
//
// WDocs are visited in alphabetic order.
func WDocs(types ITypes) func(func(IWDoc) bool) {
	return func(visit func(IWDoc) bool) {
		for t := range TypesByKind(types, TypeKind_WDoc) {
			if !visit(t.(IWDoc)) {
				break
			}
		}
	}
}

// Returns WRecord by name.
//
// Returns nil if WRecord not found.
func WRecord(types IFindType, name QName) IWRecord {
	if t := TypeByNameAndKind(types, name, TypeKind_WRecord); t != nil {
		return t.(IWRecord)
	}
	return nil
}

// Returns iterator over WRecords.
//
// WRecords are visited in alphabetic order.
func WRecords(types ITypes) func(func(IWRecord) bool) {
	return func(visit func(IWRecord) bool) {
		for t := range TypesByKind(types, TypeKind_WRecord) {
			if !visit(t.(IWRecord)) {
				break
			}
		}
	}
}

// Is specified type kind may be used in child containers.
func (k TypeKind) ContainerKindAvailable(s TypeKind) bool {
	return structTypeProps(k).containerKinds.Contains(s)
}

// Is field with data kind allowed.
func (k TypeKind) FieldKindAvailable(d DataKind) bool {
	return structTypeProps(k).fieldKinds.Contains(d)
}

// Is specified system field exists and required.
func (k TypeKind) HasSystemField(f FieldName) (exists, required bool) {
	required, exists = structTypeProps(k).systemFields[f]
	return exists, required
}

func (k TypeKind) MarshalText() ([]byte, error) {
	var s string
	if k < TypeKind_count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an TypeKind in human-readable form, without `TypeKind_` prefix,
// suitable for debugging or error messages
func (k TypeKind) TrimString() string {
	const pref = "TypeKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Structural type kind properties
type structuralTypeProps struct {
	fieldKinds     set.Set[DataKind]
	systemFields   map[FieldName]bool
	containerKinds set.Set[TypeKind]
}

var (
	nullStructProps = &structuralTypeProps{
		fieldKinds:     set.Empty[DataKind](),
		systemFields:   map[FieldName]bool{},
		containerKinds: set.Empty[TypeKind](),
	}

	structFieldKinds = set.From(
		DataKind_int32,
		DataKind_int64,
		DataKind_float32,
		DataKind_float64,
		DataKind_bytes,
		DataKind_string,
		DataKind_QName,
		DataKind_bool,
		DataKind_RecordID,
	)

	typeKindStructProps = map[TypeKind]*structuralTypeProps{
		TypeKind_GDoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:       true,
				SystemField_QName:    true,
				SystemField_IsActive: false, // exists, but not required
			},
			containerKinds: set.From(
				TypeKind_GRecord,
			),
		},
		TypeKind_CDoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:       true,
				SystemField_QName:    true,
				SystemField_IsActive: false,
			},
			containerKinds: set.From(
				TypeKind_CRecord,
			),
		},
		TypeKind_ODoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:    true,
				SystemField_QName: true,
			},
			containerKinds: set.From(
				TypeKind_ODoc, // #19322!: ODocs should be able to contain ODocs
				TypeKind_ORecord,
			),
		},
		TypeKind_WDoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:       true,
				SystemField_QName:    true,
				SystemField_IsActive: false,
			},
			containerKinds: set.From(
				TypeKind_WRecord,
			),
		},
		TypeKind_GRecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
				SystemField_IsActive:  false,
			},
			containerKinds: set.From(
				TypeKind_GRecord,
			),
		},
		TypeKind_CRecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
				SystemField_IsActive:  false,
			},
			containerKinds: set.From(
				TypeKind_CRecord,
			),
		},
		TypeKind_ORecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
			},
			containerKinds: set.From(
				TypeKind_ORecord,
			),
		},
		TypeKind_WRecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
				SystemField_IsActive:  false,
			},
			containerKinds: set.From(
				TypeKind_WRecord,
			),
		},
		TypeKind_ViewRecord: {
			fieldKinds: set.From(
				DataKind_int32,
				DataKind_int64,
				DataKind_float32,
				DataKind_float64,
				DataKind_bytes,
				DataKind_string,
				DataKind_QName,
				DataKind_bool,
				DataKind_RecordID,
				DataKind_Record,
				DataKind_Event,
			),
			systemFields: map[FieldName]bool{
				SystemField_QName: true,
			},
			containerKinds: set.Empty[TypeKind](),
		},
		TypeKind_Object: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_QName:     true,
				SystemField_Container: false, // exists, but required for nested (child) objects only
			},
			containerKinds: set.From(
				TypeKind_Object,
			),
		},
	}
)

func structTypeProps(k TypeKind) *structuralTypeProps {
	props := nullStructProps
	if p, ok := typeKindStructProps[k]; ok {
		props = p
	}
	return props
}
