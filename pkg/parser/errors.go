/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"errors"
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
)

var ErrDirContainsNoSchemaFiles = errors.New("no schema files in directory")

func ErrUnexpectedSchema(fileName, actual, expected string) error {
	return fmt.Errorf("%s: package %s; expected %s", fileName, actual, expected)
}

var ErrFunctionParamsIncorrect = errors.New("function parameters do not match")
var ErrFunctionResultIncorrect = errors.New("function result do not match")
var ErrPrimaryKeyRedeclared = errors.New("primary key redeclared")
var ErrPrimaryKeyNotDeclared = errors.New("primary key not declared")
var ErrUndefinedTableKind = errors.New("undefined table kind")
var ErrNestedTableCannotBeDocument = errors.New("nested table cannot be declared as document")
var ErrArrayFieldsNotSupportedHere = errors.New("array fields of system types not supported here")
var ErrMustBeNotNull = errors.New("field has to be NOT NULL")

// Golang: could not import github.com/alecthomas/participle/v2/asd (no required module provides package "github.com/alecthomas/participle/v2/asd")
func ErrCouldNotImport(pkgName string) error {
	return fmt.Errorf("could not import %s", pkgName)
}

func ErrUndefined(name string) error {
	return fmt.Errorf("%s undefined", name)
}

func ErrUndefinedField(name string) error {
	return fmt.Errorf("undefined field %s", name)
}

func ErrTypeNotSupported(name string) error {
	return fmt.Errorf("%s type not supported", name)
}

func ErrRedeclared(name string) error {
	return fmt.Errorf("%s redeclared", name)
}

func ErrPackageRedeclared(name string) error {
	return fmt.Errorf("package %s redeclared", name)
}

func errorAt(err error, pos *lexer.Position) error {
	return fmt.Errorf("%s: %w", pos.String(), err)
}
