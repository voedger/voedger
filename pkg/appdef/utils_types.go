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

// Returns CDoc by name.
//
// Returns nil if CDoc not found.
func CDoc(types IFindType, name QName) ICDoc {
	return TypeByNameAndKind[ICDoc](types, name, TypeKind_CDoc)
}

// Returns iterator over CDocs.
//
// CDocs are visited in alphabetic order.
func CDocs(types EnumTypes) func(func(ICDoc) bool) {
	return TypesByKind[ICDoc](types, TypeKind_CDoc)
}

// Returns Command by name.
//
// Returns nil if Command not found.
func Command(types IFindType, name QName) ICommand {
	return TypeByNameAndKind[ICommand](types, name, TypeKind_Command)
}

// Returns iterator over Commands.
//
// Command are visited in alphabetic order.
func Commands(types EnumTypes) func(func(ICommand) bool) {
	return TypesByKind[ICommand](types, TypeKind_Command)
}

// Returns CRecord by name.
//
// Returns nil if CRecord not found.
func CRecord(types IFindType, name QName) ICRecord {
	return TypeByNameAndKind[ICRecord](types, name, TypeKind_CRecord)
}

// Returns iterator over CRecords.
//
// CRecords are visited in alphabetic order.
func CRecords(types EnumTypes) func(func(ICRecord) bool) {
	return TypesByKind[ICRecord](types, TypeKind_CRecord)
}

// Returns Data type by name.
//
// Returns nil if Data not found.
func Data(types IFindType, name QName) IData {
	return TypeByNameAndKind[IData](types, name, TypeKind_Data)
}

// Returns iterator over Data types.
//
// Data types are visited in alphabetic order.
func DataTypes(types EnumTypes) func(func(IData) bool) {
	return TypesByKind[IData](types, TypeKind_Data)
}

// Returns Extension by name.
//
// Returns nil if Extension not found.
func Extension(types IFindType, name QName) IExtension {
	return TypeByName[IExtension](types, name)
}

// Returns iterator over Extensions.
//
// Extensions are visited in alphabetic order.
func Extensions(types EnumTypes) func(func(IExtension) bool) {
	return TypesByKinds[IExtension](types, TypeKind_Extensions)
}

// Returns Function by name.
//
// Returns nil if Function not found.
func Function(types IFindType, name QName) IFunction {
	return TypeByName[IFunction](types, name)
}

// Returns iterator over Functions.
//
// Functions are visited in alphabetic order.
func Functions(types EnumTypes) func(func(IFunction) bool) {
	return TypesByKinds[IFunction](types, TypeKind_Functions)
}

// Returns GDoc by name.
//
// Returns nil if GDoc not found.
func GDoc(types IFindType, name QName) IGDoc {
	return TypeByNameAndKind[IGDoc](types, name, TypeKind_GDoc)
}

// Returns iterator over GDocs.
//
// GDocs are visited in alphabetic order.
func GDocs(types EnumTypes) func(func(IGDoc) bool) {
	return TypesByKind[IGDoc](types, TypeKind_GDoc)
}

// Returns GRecord by name.
//
// Returns nil if GRecord not found.
func GRecord(types IFindType, name QName) IGRecord {
	return TypeByNameAndKind[IGRecord](types, name, TypeKind_GRecord)
}

// Returns iterator over GRecords.
//
// GRecords are visited in alphabetic order.
func GRecords(types EnumTypes) func(func(IGRecord) bool) {
	return TypesByKind[IGRecord](types, TypeKind_GRecord)
}

// Returns Job by name.
//
// Returns nil if Job not found.
func Job(types IFindType, name QName) IJob {
	return TypeByNameAndKind[IJob](types, name, TypeKind_Job)
}

// Returns iterator over Jobs.
//
// Jobs are visited in alphabetic order.
func Jobs(types EnumTypes) func(func(IJob) bool) {
	return TypesByKind[IJob](types, TypeKind_Job)
}

// Returns Limit by name.
//
// Returns nil if Limit not found.
func Limit(types IFindType, name QName) ILimit {
	return TypeByNameAndKind[ILimit](types, name, TypeKind_Limit)
}

// Returns iterator over Limits.
//
// Limits are visited in alphabetic order.
func Limits(types EnumTypes) func(func(ILimit) bool) {
	return TypesByKind[ILimit](types, TypeKind_Limit)
}

// Returns Object by name.
//
// Returns nil if Object not found.
func Object(types IFindType, name QName) IObject {
	return TypeByNameAndKind[IObject](types, name, TypeKind_Object)
}

// Returns iterator over Objects.
//
// Objects are visited in alphabetic order.
func Objects(types EnumTypes) func(func(IObject) bool) {
	return TypesByKind[IObject](types, TypeKind_Object)
}

// Returns ODoc by name.
//
// Returns nil if ODoc not found.
func ODoc(types IFindType, name QName) IODoc {
	return TypeByNameAndKind[IODoc](types, name, TypeKind_ODoc)
}

// Returns iterator over ODocs.
//
// ODocs are visited in alphabetic order.
func ODocs(types EnumTypes) func(func(IODoc) bool) {
	return TypesByKind[IODoc](types, TypeKind_ODoc)
}

// Returns ORecord by name.
//
// Returns nil if ORecord not found.
func ORecord(types IFindType, name QName) IORecord {
	return TypeByNameAndKind[IORecord](types, name, TypeKind_ORecord)
}

// Returns iterator over ORecords.
//
// ORecords are visited in alphabetic order.
func ORecords(types EnumTypes) func(func(IORecord) bool) {
	return TypesByKind[IORecord](types, TypeKind_ORecord)
}

// Returns Projector by name.
//
// Returns nil if Projector not found.
func Projector(types IFindType, name QName) IProjector {
	return TypeByNameAndKind[IProjector](types, name, TypeKind_Projector)
}

// Returns iterator over Projectors.
//
// Projectors are visited in alphabetic order.
func Projectors(types EnumTypes) func(func(IProjector) bool) {
	return TypesByKind[IProjector](types, TypeKind_Projector)
}

// Returns Query by name.
//
// Returns nil if Query not found.
func Query(types IFindType, name QName) IQuery {
	return TypeByNameAndKind[IQuery](types, name, TypeKind_Query)
}

// Returns iterator over Queries.
//
// Queries are visited in alphabetic order.
func Queries(types EnumTypes) func(func(IQuery) bool) {
	return TypesByKind[IQuery](types, TypeKind_Query)
}

// Returns Rate by name.
//
// Returns nil if Rate not found.
func Rate(types IFindType, name QName) IRate {
	return TypeByNameAndKind[IRate](types, name, TypeKind_Rate)
}

// Returns iterator over Rates.
//
// Rates are visited in alphabetic order.
func Rates(types EnumTypes) func(func(IRate) bool) {
	return TypesByKind[IRate](types, TypeKind_Rate)
}

// Returns Record by name.
//
// Returns nil if Record not found.
func Record(types IFindType, name QName) IRecord {
	return TypeByName[IRecord](types, name)
}

// Returns iterator over Records.
//
// Records are visited in alphabetic order.
func Records(types EnumTypes) func(func(IRecord) bool) {
	return TypesByKinds[IRecord](types, TypeKind_Records)
}

// Returns Role by name.
//
// Returns nil if Role not found.
func Role(types IFindType, name QName) IRole {
	return TypeByNameAndKind[IRole](types, name, TypeKind_Role)
}

// Returns iterator over Roles.
//
// Roles are visited in alphabetic order.
func Roles(types EnumTypes) func(func(IRole) bool) {
	return TypesByKind[IRole](types, TypeKind_Role)
}

// Returns Singleton by name.
//
// Returns nil if Singleton not found.
func Singleton(types IFindType, name QName) ISingleton {
	if s := TypeByName[ISingleton](types, name); (s != nil) && s.Singleton() {
		return s
	}
	return nil
}

// Returns iterator over Singletons.
//
// Singletons are visited in alphabetic order.
func Singletons(types EnumTypes) func(func(ISingleton) bool) {
	return func(visit func(ISingleton) bool) {
		for s := range TypesByKinds[ISingleton](types, TypeKind_Singletons) {
			if s.Singleton() {
				if !visit(s) {
					break
				}
			}
		}
	}
}

// Returns Structure by name.
//
// Returns nil if Structure not found.
func Structure(types IFindType, name QName) IStructure {
	return TypeByName[IStructure](types, name)
}

// Returns iterator over Structures.
//
// Structures are visited in alphabetic order.
func Structures(types EnumTypes) func(func(IStructure) bool) {
	return TypesByKinds[IStructure](types, TypeKind_Structures)
}

// Returns system Data type (sys.int32, sys.float654, etc.) by data kind.
//
// Returns nil if not found.
func SysData(types IFindType, k DataKind) IData {
	return TypeByNameAndKind[IData](types, SysDataName(k), TypeKind_Data)
}

// Returns type by name.
//
// Returns nil if type not found.
func TypeByName[T IType](types IFindType, name QName) (found T) {
	if t := types.Type(name); t != NullType {
		if r, ok := t.(T); ok {
			found = r
		}
	}
	return found
}

// Returns type by name and kind.
//
// Returns nil if type not found.
func TypeByNameAndKind[T IType](types IFindType, name QName, kind TypeKind) (found T) {
	if t := types.Type(name); t.Kind() == kind {
		found = t.(T)
	}
	return found
}

// Returns iterator over types by kind.
//
// Types are visited in alphabetic order.
func TypesByKind[T IType](types EnumTypes, kind TypeKind) func(func(T) bool) {
	return func(visit func(T) bool) {
		for t := range types {
			if t.Kind() == kind {
				if !visit(t.(T)) {
					break
				}
			}
		}
	}
}

// Returns iterator over types by kinds set.
//
// Types are visited in alphabetic order.
func TypesByKinds[T IType](types EnumTypes, kinds TypeKindSet) func(func(T) bool) {
	return func(visit func(T) bool) {
		for t := range types {
			if kinds.Contains(t.Kind()) {
				if !visit(t.(T)) {
					break
				}
			}
		}
	}
}

// Returns View by name.
//
// Returns nil if View not found.
func View(types IFindType, name QName) IView {
	return TypeByNameAndKind[IView](types, name, TypeKind_ViewRecord)
}

// Returns iterator over Views.
//
// Views are visited in alphabetic order.
func Views(types EnumTypes) func(func(IView) bool) {
	return TypesByKind[IView](types, TypeKind_ViewRecord)
}

// Returns WDoc by name.
//
// Returns nil if WDoc not found.
func WDoc(types IFindType, name QName) IWDoc {
	return TypeByNameAndKind[IWDoc](types, name, TypeKind_WDoc)
}

// Returns iterator over WDocs.
//
// WDocs are visited in alphabetic order.
func WDocs(types EnumTypes) func(func(IWDoc) bool) {
	return TypesByKind[IWDoc](types, TypeKind_WDoc)
}

// Returns WRecord by name.
//
// Returns nil if WRecord not found.
func WRecord(types IFindType, name QName) IWRecord {
	return TypeByNameAndKind[IWRecord](types, name, TypeKind_WRecord)
}

// Returns iterator over WRecords.
//
// WRecords are visited in alphabetic order.
func WRecords(types EnumTypes) func(func(IWRecord) bool) {
	return TypesByKind[IWRecord](types, TypeKind_WRecord)
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
