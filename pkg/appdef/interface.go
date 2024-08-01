/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Application definition is a set of types, views, commands, queries and workspaces.
type IAppDef interface {
	IWithComments

	IWithPackages
	IWithWorkspaces

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
	IWithFunctions
	IWithCommands
	IWithQueries
	IWithProjectors
	IWithJobs

	IWithRoles
	IWithPrivileges

	IWithRates
	IWithLimits
}

type IAppDefBuilder interface {
	ICommentsBuilder

	IPackagesBuilder
	IWorkspacesBuilder

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
	IJobsBuilder

	IRolesBuilder
	IPrivilegesBuilder

	IRatesBuilder
	ILimitsBuilder

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

	// Builds application definition.
	//
	// Validates and returns builded application type.
	// Must be called after all entities added.
	//
	// # Panics if error occurred.
	MustBuild() IAppDef
}
