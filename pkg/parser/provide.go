/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

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

// MergeFileSchemaASTs merges File Schema ASTs into Package Schema AST.
// Performs package-level semantic analysis
func MergeFileSchemaASTs(qualifiedPackageName string, asts []*FileSchemaAST) (*PackageSchemaAST, error) {
	return mergeFileSchemaASTsImpl(qualifiedPackageName, asts)
}

// ParsePackageDir is a helper which parses all SQL schemas from specified FS and returns Package Schema.
func ParsePackageDir(qualifiedPackageName string, fs IReadFS, subDir string) (*PackageSchemaAST, error) {
	asts, err := parseFSImpl(fs, subDir)
	if err != nil {
		return nil, err
	}
	return MergeFileSchemaASTs(qualifiedPackageName, asts)
}

// Application-level semantic analysis (e.g. cross-package references)
func MergePackageSchemas([]*PackageSchemaAST) /*TODO: return .?.ISchema */ error {
	// TODO: implement
	return nil
}
