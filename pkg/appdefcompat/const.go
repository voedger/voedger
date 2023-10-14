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
	ConstraintChangeAllowed Constraint = "ConstraintChangeAllowed"
)

const (
	ErrorTypeNodeRemoved  ErrorType = "NodeRemoved"
	ErrorTypeOrderChanged ErrorType = "OrderChanged"
	ErrorTypeNodeInserted ErrorType = "NodeInserted"
	ErrorTypeValueChanged ErrorType = "ValueChanged"
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
	NodeNameArgs            = "Args"
	NodeNameResult          = "Result"
	NodeNameInherits        = "Inherits"
	NodeNameUnloggedArgs    = "UnloggedArgs"
)
