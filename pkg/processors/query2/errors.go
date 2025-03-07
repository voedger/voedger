/*
 * Copyright (c) 2025-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import "errors"

var (
	errConstraintsAreNull                        = errors.New("constraints are null")
	errWhereConstraintIsEmpty                    = errors.New("where constraint is empty")
	errWhereConstraintMustSpecifyThePartitionKey = errors.New("where constraint must specify the partition key")
	errUnsupportedConstraint                     = errors.New("unsupported constraint")
	errUnexpectedParams                          = errors.New("unexpected params")
	errUnsupportedType                           = errors.New("unsupported type")
	errUnexpectedField                           = errors.New("unexpected field")
)
