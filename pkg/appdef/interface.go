/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Application.
//
// Ref to apdef.go for implementation
type IAppDef interface {
	IComment

	// Returns type by name.
	//
	// If not found then empty type with TypeKind_null is returned
	Type(name QName) IType

	// Returns type by name.
	//
	// Returns nil if type not found.
	TypeByName(name QName) IType

	// Return count of types.
	TypeCount() int

	// Enumerates all application types.
	//
	// Types are enumerated in alphabetical order by QName
	Types(func(IType))

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

	// Return Element by name.
	//
	// Returns nil if not found.
	Element(name QName) IElement

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

	// Returns workspace by name.
	//
	// Returns nil if not found.
	Workspace(QName) IWorkspace
}

// Application builder
//
// Ref to appdef.go for implementation
type IAppDefBuilder interface {
	IAppDef
	ICommentBuilder

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

	// Adds new singleton CDoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddSingleton(name QName) ICDocBuilder

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

	// Adds new Element type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddElement(name QName) IElementBuilder

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

	// Adds new workspace.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddWorkspace(QName) IWorkspaceBuilder

	// Must be called after all types added. Validates and returns builded application type or error
	Build() (IAppDef, error)
}
