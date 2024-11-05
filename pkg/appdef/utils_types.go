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

// Returns Extension by name.
//
// Returns nil if Extension not found.
func Extension(types IFindType, name QName) IExtension {
	if t := TypeByName(types, name); t != nil {
		if r, ok := t.(IExtension); ok {
			return r
		}
	}
	return nil
}

// Returns iterator over Extensions.
//
// Extensions are visited in alphabetic order.
func Extensions(types ITypes) func(func(IExtension) bool) {
	return func(visit func(IExtension) bool) {
		for t := range TypesByKinds(types, TypeKind_Extensions) {
			if !visit(t.(IExtension)) {
				break
			}
		}
	}
}

// Returns Function by name.
//
// Returns nil if Function not found.
func Function(types IFindType, name QName) IFunction {
	if t := TypeByName(types, name); t != nil {
		if r, ok := t.(IFunction); ok {
			return r
		}
	}
	return nil
}

// Returns iterator over Functions.
//
// Functions are visited in alphabetic order.
func Functions(types ITypes) func(func(IFunction) bool) {
	return func(visit func(IFunction) bool) {
		for t := range TypesByKinds(types, TypeKind_Functions) {
			if !visit(t.(IFunction)) {
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

// Returns Job by name.
//
// Returns nil if Job not found.
func Job(types IFindType, name QName) IJob {
	if t := TypeByNameAndKind(types, name, TypeKind_Job); t != nil {
		return t.(IJob)
	}
	return nil
}

// Returns iterator over Jobs.
//
// Jobs are visited in alphabetic order.
func Jobs(types ITypes) func(func(IJob) bool) {
	return func(visit func(IJob) bool) {
		for t := range TypesByKind(types, TypeKind_Job) {
			if !visit(t.(IJob)) {
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

// Returns Projector by name.
//
// Returns nil if Projector not found.
func Projector(types IFindType, name QName) IProjector {
	if t := TypeByNameAndKind(types, name, TypeKind_Projector); t != nil {
		return t.(IProjector)
	}
	return nil
}

// Returns iterator over Projectors.
//
// Projectors are visited in alphabetic order.
func Projectors(types ITypes) func(func(IProjector) bool) {
	return func(visit func(IProjector) bool) {
		for t := range TypesByKind(types, TypeKind_Projector) {
			if !visit(t.(IProjector)) {
				break
			}
		}
	}
}

// Returns Query by name.
//
// Returns nil if Query not found.
func Query(types IFindType, name QName) IQuery {
	if t := TypeByNameAndKind(types, name, TypeKind_Query); t != nil {
		return t.(IQuery)
	}
	return nil
}

// Returns iterator over Queries.
//
// Queries are visited in alphabetic order.
func Queries(types ITypes) func(func(IQuery) bool) {
	return func(visit func(IQuery) bool) {
		for t := range TypesByKind(types, TypeKind_Query) {
			if !visit(t.(IQuery)) {
				break
			}
		}
	}
}

// Returns Record by name.
//
// Returns nil if Record not found.
func Record(types IFindType, name QName) IRecord {
	if t := TypeByName(types, name); t != nil {
		if r, ok := t.(IRecord); ok {
			return r
		}
	}
	return nil
}

// Returns iterator over Records.
//
// Records are visited in alphabetic order.
func Records(types ITypes) func(func(IRecord) bool) {
	return func(visit func(IRecord) bool) {
		for t := range TypesByKinds(types, TypeKind_Records) {
			if !visit(t.(IRecord)) {
				break
			}
		}
	}
}

// Returns Role by name.
//
// Returns nil if Role not found.
func Role(types IFindType, name QName) IRole {
	if t := TypeByNameAndKind(types, name, TypeKind_Role); t != nil {
		return t.(IRole)
	}
	return nil
}

// Returns iterator over Roles.
//
// Roles are visited in alphabetic order.
func Roles(types ITypes) func(func(IRole) bool) {
	return func(visit func(IRole) bool) {
		for t := range TypesByKind(types, TypeKind_Role) {
			if !visit(t.(IRole)) {
				break
			}
		}
	}
}

// Returns Singleton by name.
//
// Returns nil if Singleton not found.
func Singleton(types IFindType, name QName) ISingleton {
	if t := TypeByName(types, name); t != nil {
		if s, ok := t.(ISingleton); ok {
			if s.Singleton() {
				return s
			}
		}
	}
	return nil
}

// Returns iterator over Singletons.
//
// Singletons are visited in alphabetic order.
func Singletons(types ITypes) func(func(ISingleton) bool) {
	return func(visit func(ISingleton) bool) {
		for t := range TypesByKinds(types, TypeKind_Singletons) {
			if s, ok := t.(ISingleton); ok {
				if s.Singleton() {
					if !visit(s) {
						break
					}
				}
			}
		}
	}
}

// Returns Structure by name.
//
// Returns nil if Structure not found.
func Structure(types IFindType, name QName) IStructure {
	if t := TypeByName(types, name); t != nil {
		if s, ok := t.(IStructure); ok {
			return s
		}
	}
	return nil
}

// Returns iterator over Structures.
//
// Structures are visited in alphabetic order.
func Structures(types ITypes) func(func(IStructure) bool) {
	return func(visit func(IStructure) bool) {
		for t := range TypesByKinds(types, TypeKind_Structures) {
			if !visit(t.(IStructure)) {
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

// Returns View by name.
//
// Returns nil if View not found.
func View(types IFindType, name QName) IView {
	if t := TypeByNameAndKind(types, name, TypeKind_ViewRecord); t != nil {
		return t.(IView)
	}
	return nil
}

// Returns iterator over Views.
//
// Views are visited in alphabetic order.
func Views(types ITypes) func(func(IView) bool) {
	return func(visit func(IView) bool) {
		for t := range TypesByKind(types, TypeKind_ViewRecord) {
			if !visit(t.(IView)) {
				break
			}
		}
	}
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
