/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package appdefcompat

const (
	validationErrorFmt = "%s: %s"
	pathDelimiter      = "/"
)

const (
	ConstraintValueMatch Constraint = 1 << iota
	ConstraintAppendOnly
	ConstraintInsertOnly
	ConstraintDeleteOnly
	ConstraintOrderChangeOnly
	ConstraintAllAllowed    = 255
	ConstraintNonModifiable = 0
)

const (
	ErrorTypeNodeRemoved  ErrorType = "NodeRemoved"
	ErrorTypeOrderChanged ErrorType = "OrderChanged"
	ErrorTypeNodeInserted ErrorType = "NodeInserted"
	ErrorTypeValueChanged ErrorType = "ValueChanged"
	ErrorTypeNodeModified ErrorType = "NodeModified"
)

const (
	NodeNameTypes           = "Types"
	NodeNameFields          = "Fields"
	NodeNameUniques         = "Uniques"
	NodeNameUniqueFields    = "UniqueFields"
	NodeNameAbstract        = "Abstract"
	NodeNameContainers      = "Containers"
	NodeNameAppDef          = "AppDef"
	NodeNameDescriptor      = "Descriptor"
	NodeNamePartKeyFields   = "PartKeyFields"
	NodeNameClustColsFields = "ClustColsFields"
	NodeNameCommandArgs     = "CommandArgs"
	NodeNameCommandResult   = "CommandResult"
	NodeNameQueryArgs       = "QueryArgs"
	NodeNameQueryResult     = "QueryResult"
	NodeNameUnloggedArgs    = "UnloggedArgs"
	NodeNamePackages        = "Packages"
)
