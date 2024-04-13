/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Application definition is a set of types, views, commands, queries and workspaces.
type IAppDef interface {
	IWithComment
	IWithPackages

	IWithTypes
	IWithDataTypes

	IWithStructures
	IWithRecords
	IWithGDocs
	IWithCDocs
	IWithWDocs
	IWithSingletons
	IWithODocs
	IWithObjects

	IWithViews

	IWithExtensions
	IWithCommands
	IWithQueries
	IWithProjectors

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
	IDataTypesBuilder
	IGDocsBuilder
	ICDocsBuilder
	IWDocsBuilder
	IODocsBuilder
	IObjectsBuilder
	IViewsBuilder
	ICommandsBuilder
	IQueriesBuilder
	IProjectorsBuilder

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
