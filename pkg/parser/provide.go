/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// ParseFile parses content of the single file, creates FileSchemaAST and returns pointer to it.
// Performs syntax analysis
func ParseFile(fileName, content string) (*FileSchemaAST, error) {
	ast, err := parseImpl(fileName, content)
	if err != nil {
		return nil, err
	}
	return &FileSchemaAST{
		FileName: fileName,
		Ast:      ast,
	}, nil
}

// BuildPackageSchema merges File Schema ASTs into Package Schema AST.
// Performs package-level semantic analysis
func BuildPackageSchema(qualifiedPackageName string, asts []*FileSchemaAST) (*PackageSchemaAST, error) {
	return buildPackageSchemaImpl(qualifiedPackageName, asts)
}

// ParsePackageDir is a helper which parses all SQL schemas from specified FS and returns Package Schema.
func ParsePackageDir(qualifiedPackageName string, fs IReadFS, subDir string) (*PackageSchemaAST, error) {
	asts, errs := parseFSImpl(fs, subDir)
	packageAst, packageBuildErr := BuildPackageSchema(qualifiedPackageName, asts)
	if packageBuildErr != nil {
		errs = append(errs, coreutils.SplitErrors(packageBuildErr)...)
	}
	return packageAst, errors.Join(errs...)
}

// Application-level semantic analysis (e.g. cross-package references)
func BuildAppSchema(packages []*PackageSchemaAST) (*AppSchemaAST, error) {
	return buildAppSchemaImpl(packages)
}

func BuildAppDefs(appSchema *AppSchemaAST, builder appdef.IAppDefBuilder) error {
	return buildAppDefs(appSchema, builder)
}
