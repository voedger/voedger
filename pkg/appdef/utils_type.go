/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"iter"
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/set"
)

// Returns CDoc by name.
//
// Returns nil if CDoc not found.
func CDoc(f FindType, name QName) ICDoc {
	return TypeByNameAndKind[ICDoc](f, name, TypeKind_CDoc)
}

// Returns iterator over CDocs.
func CDocs(types TypesSlice) iter.Seq[ICDoc] {
	return TypesByKind[ICDoc](types, TypeKind_CDoc)
}

// Returns Command by name.
//
// Returns nil if Command not found.
func Command(f FindType, name QName) ICommand {
	return TypeByNameAndKind[ICommand](f, name, TypeKind_Command)
}

// Returns iterator over Commands.
func Commands(types TypesSlice) iter.Seq[ICommand] {
	return TypesByKind[ICommand](types, TypeKind_Command)
}

// Returns CRecord by name.
//
// Returns nil if CRecord not found.
func CRecord(f FindType, name QName) ICRecord {
	return TypeByNameAndKind[ICRecord](f, name, TypeKind_CRecord)
}

// Returns iterator over CRecords.
func CRecords(types TypesSlice) iter.Seq[ICRecord] {
	return TypesByKind[ICRecord](types, TypeKind_CRecord)
}

// Returns Data type by name.
//
// Returns nil if Data not found.
func Data(f FindType, name QName) IData {
	return TypeByNameAndKind[IData](f, name, TypeKind_Data)
}

// Returns iterator over Data types.
func DataTypes(types TypesSlice) iter.Seq[IData] {
	return TypesByKind[IData](types, TypeKind_Data)
}

// Returns Extension by name.
//
// Returns nil if Extension not found.
func Extension(f FindType, name QName) IExtension {
	return TypeByName[IExtension](f, name)
}

// Returns iterator over Extensions.
func Extensions(types TypesSlice) iter.Seq[IExtension] {
	return TypesByKinds[IExtension](types, TypeKind_Extensions)
}

// Returns Function by name.
//
// Returns nil if Function not found.
func Function(f FindType, name QName) IFunction {
	return TypeByName[IFunction](f, name)
}

// Returns iterator over Functions.
func Functions(types TypesSlice) iter.Seq[IFunction] {
	return TypesByKinds[IFunction](types, TypeKind_Functions)
}

// Returns GDoc by name.
//
// Returns nil if GDoc not found.
func GDoc(f FindType, name QName) IGDoc {
	return TypeByNameAndKind[IGDoc](f, name, TypeKind_GDoc)
}

// Returns iterator over GDocs.
func GDocs(types TypesSlice) iter.Seq[IGDoc] {
	return TypesByKind[IGDoc](types, TypeKind_GDoc)
}

// Returns GRecord by name.
//
// Returns nil if GRecord not found.
func GRecord(f FindType, name QName) IGRecord {
	return TypeByNameAndKind[IGRecord](f, name, TypeKind_GRecord)
}

// Returns iterator over GRecords.
func GRecords(types TypesSlice) iter.Seq[IGRecord] {
	return TypesByKind[IGRecord](types, TypeKind_GRecord)
}

// Returns Job by name.
//
// Returns nil if Job not found.
func Job(f FindType, name QName) IJob {
	return TypeByNameAndKind[IJob](f, name, TypeKind_Job)
}

// Returns iterator over Jobs.
func Jobs(types TypesSlice) iter.Seq[IJob] {
	return TypesByKind[IJob](types, TypeKind_Job)
}

// Returns Limit by name.
//
// Returns nil if Limit not found.
func Limit(f FindType, name QName) ILimit {
	return TypeByNameAndKind[ILimit](f, name, TypeKind_Limit)
}

// Returns iterator over Limits.
func Limits(types TypesSlice) iter.Seq[ILimit] {
	return TypesByKind[ILimit](types, TypeKind_Limit)
}

// Returns Object by name.
//
// Returns nil if Object not found.
func Object(f FindType, name QName) IObject {
	return TypeByNameAndKind[IObject](f, name, TypeKind_Object)
}

// Returns iterator over Objects.
func Objects(types TypesSlice) iter.Seq[IObject] {
	return TypesByKind[IObject](types, TypeKind_Object)
}

// Returns ODoc by name.
//
// Returns nil if ODoc not found.
func ODoc(f FindType, name QName) IODoc {
	return TypeByNameAndKind[IODoc](f, name, TypeKind_ODoc)
}

// Returns iterator over ODocs.
func ODocs(types TypesSlice) iter.Seq[IODoc] {
	return TypesByKind[IODoc](types, TypeKind_ODoc)
}

// Returns ORecord by name.
//
// Returns nil if ORecord not found.
func ORecord(f FindType, name QName) IORecord {
	return TypeByNameAndKind[IORecord](f, name, TypeKind_ORecord)
}

// Returns iterator over ORecords.
func ORecords(types TypesSlice) iter.Seq[IORecord] {
	return TypesByKind[IORecord](types, TypeKind_ORecord)
}

// Returns Projector by name.
//
// Returns nil if Projector not found.
func Projector(f FindType, name QName) IProjector {
	return TypeByNameAndKind[IProjector](f, name, TypeKind_Projector)
}

// Returns iterator over Projectors.
func Projectors(types TypesSlice) iter.Seq[IProjector] {
	return TypesByKind[IProjector](types, TypeKind_Projector)
}

// Returns Query by name.
//
// Returns nil if Query not found.
func Query(f FindType, name QName) IQuery {
	return TypeByNameAndKind[IQuery](f, name, TypeKind_Query)
}

// Returns iterator over Queries.
func Queries(types TypesSlice) iter.Seq[IQuery] {
	return TypesByKind[IQuery](types, TypeKind_Query)
}

// Returns Rate by name.
//
// Returns nil if Rate not found.
func Rate(f FindType, name QName) IRate {
	return TypeByNameAndKind[IRate](f, name, TypeKind_Rate)
}

// Returns iterator over Rates.
func Rates(types TypesSlice) iter.Seq[IRate] {
	return TypesByKind[IRate](types, TypeKind_Rate)
}

// Returns Record by name.
//
// Returns nil if Record not found.
func Record(f FindType, name QName) IRecord {
	return TypeByName[IRecord](f, name)
}

// Returns iterator over Records.
func Records(types TypesSlice) iter.Seq[IRecord] {
	return TypesByKinds[IRecord](types, TypeKind_Records)
}

// Returns Role by name.
//
// Returns nil if Role not found.
func Role(f FindType, name QName) IRole {
	return TypeByNameAndKind[IRole](f, name, TypeKind_Role)
}

// Returns iterator over Roles.
func Roles(types TypesSlice) iter.Seq[IRole] {
	return TypesByKind[IRole](types, TypeKind_Role)
}

// Returns Singleton by name.
//
// Returns nil if Singleton not found.
func Singleton(f FindType, name QName) ISingleton {
	if s := TypeByName[ISingleton](f, name); (s != nil) && s.Singleton() {
		return s
	}
	return nil
}

// Returns iterator over Singletons.
func Singletons(types TypesSlice) iter.Seq[ISingleton] {
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
func Structure(f FindType, name QName) IStructure {
	return TypeByName[IStructure](f, name)
}

// Returns iterator over Structures.
func Structures(types TypesSlice) iter.Seq[IStructure] {
	return TypesByKinds[IStructure](types, TypeKind_Structures)
}

// Returns system Data type (sys.int32, sys.float654, etc.) by data kind.
//
// Returns nil if not found.
func SysData(f FindType, k DataKind) IData {
	return TypeByNameAndKind[IData](f, SysDataName(k), TypeKind_Data)
}

// Returns Tag by name.
//
// Returns nil if Tag not found.
func Tag(f FindType, name QName) ITag {
	return TypeByNameAndKind[ITag](f, name, TypeKind_Tag)
}

// Returns iterator over Tags.
func Tags(types TypesSlice) iter.Seq[ITag] {
	return TypesByKind[ITag](types, TypeKind_Tag)
}

// Returns type by name.
//
// Returns nil if type not found.
func TypeByName[T IType](f FindType, name QName) (found T) {
	if t := f(name); t != NullType {
		if r, ok := t.(T); ok {
			found = r
		}
	}
	return found
}

// Returns type by name and kind.
//
// Returns nil if type not found.
func TypeByNameAndKind[T IType](f FindType, name QName, kind TypeKind) (found T) {
	if t := f(name); t.Kind() == kind {
		found = t.(T)
	}
	return found
}

// Returns iterator over types by kind.
func TypesByKind[T IType](types TypesSlice, kind TypeKind) iter.Seq[T] {
	return func(visit func(T) bool) {
		for _, t := range types {
			if t.Kind() == kind {
				if !visit(t.(T)) {
					break
				}
			}
		}
	}
}

// Returns iterator over types by kinds set.
func TypesByKinds[T IType](types TypesSlice, kinds TypeKindSet) iter.Seq[T] {
	return func(visit func(T) bool) {
		for _, t := range types {
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
func View(f FindType, name QName) IView {
	return TypeByNameAndKind[IView](f, name, TypeKind_ViewRecord)
}

// Returns iterator over Views.
func Views(types TypesSlice) iter.Seq[IView] {
	return TypesByKind[IView](types, TypeKind_ViewRecord)
}

// Returns WDoc by name.
//
// Returns nil if WDoc not found.
func WDoc(f FindType, name QName) IWDoc {
	return TypeByNameAndKind[IWDoc](f, name, TypeKind_WDoc)
}

// Returns iterator over WDocs.
func WDocs(types TypesSlice) iter.Seq[IWDoc] {
	return TypesByKind[IWDoc](types, TypeKind_WDoc)
}

// Returns WRecord by name.
//
// Returns nil if WRecord not found.
func WRecord(f FindType, name QName) IWRecord {
	return TypeByNameAndKind[IWRecord](f, name, TypeKind_WRecord)
}

// Returns iterator over WRecords.
func WRecords(types TypesSlice) iter.Seq[IWRecord] {
	return TypesByKind[IWRecord](types, TypeKind_WRecord)
}

type anyType struct {
	nullComment
	nullTags
	name QName
}

func newAnyType(name QName) IType { return &anyType{nullComment{}, nullTags{}, name} }

func (t anyType) App() IAppDef          { return nil }
func (t anyType) IsSystem() bool        { return true }
func (t anyType) Kind() TypeKind        { return TypeKind_Any }
func (t anyType) QName() QName          { return t.name }
func (t anyType) String() string        { return t.name.Entity() + " type" }
func (t anyType) Workspace() IWorkspace { return nil }

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
