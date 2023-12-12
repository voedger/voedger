/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import "github.com/voedger/voedger/pkg/parser"

type vpmParams struct {
	WorkingDir string
	TargetDir  string
	IgnoreFile string
}

// packageFiles is a map of package name to a list of files that belong to the package
type packageFiles map[string][]string

// compileResult is a result of compilation of a single module
type compileResult struct {
	modulePath   string               // module path of compiled module
	pkgFiles     packageFiles         // files that belong to the module
	appSchemaAST *parser.AppSchemaAST // app schema of the compiled module
}

// baselineInfo is a struct that is saved to baseline.json file
type baselineInfo struct {
	BaselinePackageUrl string `json:"BaselinePackageUrl"`
	Timestamp          string `json:"Timestamp"`
	GitCommitHash      string `json:"GitCommitHash,omitempty"`
}

type ignoreInfo struct {
	Ignore []string `yaml:"Ignore"`
}
