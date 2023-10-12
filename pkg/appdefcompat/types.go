/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"errors"
	"fmt"
	"strings"
)

type Constraint string
type ErrorType string

type CompatibilityTreeNode struct {
	Name       string
	Props      []*CompatibilityTreeNode
	Value      interface{}
	ParentNode *CompatibilityTreeNode
}

func (n *CompatibilityTreeNode) Path() []string {
	if n.ParentNode == nil {
		return []string{n.Name}
	}
	return append(n.ParentNode.Path(), n.Name)
}

type NodeConstraint struct {
	NodeName   string
	Constraint Constraint
}

type CompatibilityError struct {
	Constraint  Constraint
	OldTreePath []string
	ErrorType   ErrorType
}

func newCompatibilityError(constraint Constraint, oldTreePath []string, errType ErrorType) CompatibilityError {
	return CompatibilityError{
		Constraint:  constraint,
		OldTreePath: oldTreePath,
		ErrorType:   errType,
	}
}

func (e CompatibilityError) Error() string {
	return fmt.Sprintf(validationErrorFmt, e.ErrorType, e.Path())
}

func (e CompatibilityError) Path() string {
	return strings.Join(e.OldTreePath, pathDelimiter)
}

type CompatibilityErrors struct {
	Errors []CompatibilityError
}

func (e *CompatibilityErrors) Error() string {
	errs := make([]error, len(e.Errors))
	for i, err := range e.Errors {
		errs[i] = err
	}
	return errors.Join(errs...).Error()
}

// matchNodesResult represents the result of matching nodes.
type matchNodesResult struct {
	InsertedNodeCount  int
	DeletedNodeNames   []string
	AppendedNodeCount  int
	MatchedNodePairs   [][2]*CompatibilityTreeNode
	ReorderedNodeNames []string
}
