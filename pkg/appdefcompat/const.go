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
	NodeRemoved  NodeErrorString = "NodeRemoved"
	OrderChanged NodeErrorString = "OrderChanged"
	NodeInserted NodeErrorString = "NodeInserted"
	ValueChanged NodeErrorString = "ValueChanged"
)
