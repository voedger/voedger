/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

const (
	validationErrorFmt = "%s: %s"
	pathDelimiter      = "->"
)

const (
	ConstraintValueMatch    Constraint = "ConstraintValueMatch"
	ConstraintAppendOnly    Constraint = "ConstraintAppendOnly"
	ConstraintInsertOnly    Constraint = "ConstraintInsertOnly"
	ConstraintNonModifiable Constraint = "ConstraintNonModifiable"
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
	NodeNameUniqueFields    = "UniqueFields"
	NodeNameAbstract        = "Abstract"
	NodeNameParent          = "Parent"
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
)
