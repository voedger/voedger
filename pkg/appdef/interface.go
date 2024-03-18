/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Application definition is a set of types, views, commands, queries and workspaces.
type IAppDef interface {
	IComment
	IWithTypes
	IWithPackages

	// Return data type by name.
	//
	// Returns nil if not found.
	Data(name QName) IData

	// Enumerates all application data types.
	//
	// If incSys specified then system data types are included into enumeration.
	//
	// Data types are enumerated in alphabetical order by QName.
	DataTypes(incSys bool, cb func(IData))

	// Returns system data type (sys.int32, sys.float654, etc.) by data kind.
	//
	// Returns nil if not found.
	SysData(DataKind) IData

	// Return GDoc by name.
	//
	// Returns nil if not found.
	GDoc(name QName) IGDoc

	// Return GRecord by name.
	//
	// Returns nil if not found.
	GRecord(name QName) IGRecord

	// Return CDoc by name.
	//
	// Returns nil if not found.
	CDoc(name QName) ICDoc

	// Return CRecord by name.
	//
	// Returns nil if not found.
	CRecord(name QName) ICRecord

	// Return WDoc by name.
	//
	// Returns nil if not found.
	WDoc(name QName) IWDoc

	// Return WRecord by name.
	//
	// Returns nil if not found.
	WRecord(name QName) IWRecord

	// Return ODoc by name.
	//
	// Returns nil if not found.
	ODoc(name QName) IODoc

	// Return ORecord by name.
	//
	// Returns nil if not found.
	ORecord(name QName) IORecord

	// Return Object by name.
	//
	// Returns nil if not found.
	Object(name QName) IObject

	// Return record by name.
	//
	// Returns nil if not found.
	Record(QName) IRecord

	// Enumerates all application records, e.g. documents and contained records
	//
	// Records are enumerated in alphabetical order by QName
	Records(func(IRecord))

	// Enumerates all application structures
	//
	// Structures are enumerated in alphabetical order by QName
	Structures(func(IStructure))

	// Return View by name.
	//
	// Returns nil if not found.
	View(name QName) IView

	// Returns Command by name.
	//
	// Returns nil if not found.
	Command(QName) ICommand

	// Returns Query by name.
	//
	// Returns nil if not found.
	Query(QName) IQuery

	// Return projector by name.
	//
	// Returns nil if not found.
	Projector(QName) IProjector

	// Enumerates all application projectors.
	//
	// Projectors are enumerated in alphabetical order by QName.
	Projectors(func(IProjector))

	// Enumerates all application extensions (commands, queries and extensions)
	//
	// Extensions are enumerated in alphabetical order by QName
	Extensions(func(IExtension))

	// Returns workspace by name.
	//
	// Returns nil if not found.
	Workspace(QName) IWorkspace

	// Returns workspace by descriptor.
	//
	// Returns nil if not found.
	WorkspaceByDescriptor(QName) IWorkspace
}

type IAppDefBuilder interface {
	ICommentBuilder
	IPackagesBuilder

	// Adds new data type with specified name and kind.
	//
	// If ancestor is not empty, then new data type inherits from.
	// If ancestor is empty, then new data type inherits from system data types with same data kind.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists,
	//   - if ancestor is not found,
	//	 - if ancestor is not data,
	//	 - if ancestor has different kind,
	//	 - if constraints are not compatible with data kind.
	AddData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder

	// Adds new GDoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddGDoc(name QName) IGDocBuilder

	// Adds new GRecord type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddGRecord(name QName) IGRecordBuilder

	// Adds new CDoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddCDoc(name QName) ICDocBuilder

	// Adds new CRecord type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddCRecord(name QName) ICRecordBuilder

	// Adds new WDoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddWDoc(name QName) IWDocBuilder

	// Adds new WRecord type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddWRecord(name QName) IWRecordBuilder

	// Adds new ODoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddODoc(name QName) IODocBuilder

	// Adds new ORecord type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddORecord(name QName) IORecordBuilder

	// Adds new Object type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddObject(name QName) IObjectBuilder

	// Adds new types for view.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddView(QName) IViewBuilder

	// Adds new command.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddCommand(QName) ICommandBuilder

	// Adds new query.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddQuery(QName) IQueryBuilder

	// Adds new projector.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddProjector(QName) IProjectorBuilder

	// Adds new workspace.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddWorkspace(QName) IWorkspaceBuilder

	// Returns application definition while building.
	//
	// Can be called before or after all entities added.
	// Does not validate application definition, some types may be invalid.
	AppDef() IAppDef

	// Builds application definition.
	//
	// Validates and returns builded application type or error.
	// Must be called after all entities added.
	Build() (IAppDef, error)
}
