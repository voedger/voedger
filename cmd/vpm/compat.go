/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdefcompat"
	"github.com/voedger/voedger/pkg/parser"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func newCompatCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "compat",
		Short: "check backward compatibility",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newParams, err := setUpParams(params, args)
			if err != nil {
				return err
			}
			ignores, err := readIgnoreFile(newParams.IgnoreFile)
			if err != nil {
				return err
			}
			compileRes, err := compile(newParams.WorkingDir)
			if err != nil {
				return err
			}
			return compat(compileRes, newParams, ignores)
		},
	}
	initGlobalFlags(cmd, &params)
	cmd.Flags().StringVarP(&params.IgnoreFile, "ignore", "", "", "path to yaml file which contains list of errors to be ignored")
	return cmd
}

// compat checks compatibility of schemas in working dir with baseline schemas in target dir
func compat(compileRes *compileResult, params vpmParams, ignores [][]string) error {
	baselineDir := params.TargetDir
	var errs []error
	baselineAppDef, err := appDefFromBaselineDir(baselineDir)
	if err != nil {
		errs = append(errs, coreutils.SplitErrors(err)...)
	}

	compiledAppDef, err := appDefFromCompiled(compileRes)
	if err != nil {
		errs = append(errs, coreutils.SplitErrors(err)...)
	}

	compatErrs := appdefcompat.CheckBackwardCompatibility(baselineAppDef, compiledAppDef)
	compatErrs = appdefcompat.IgnoreCompatibilityErrors(compatErrs, ignores)
	if len(compatErrs.Errors) > 0 {
		errObjs := make([]error, len(compatErrs.Errors))
		for i, err := range compatErrs.Errors {
			errObjs[i] = err
		}
		errs = append(errs, errObjs...)
		return errors.Join(errs...)
	}
	return nil
}

// readIgnoreFile reads yaml file and returns list of errors to be ignored
func readIgnoreFile(ignoreFilePath string) ([][]string, error) {
	if ignoreFilePath != "" {
		content, err := os.ReadFile(ignoreFilePath)
		if err != nil {
			return nil, err
		}

		var ignoreInfoObj ignoreInfo
		if err := yaml.Unmarshal(content, &ignoreInfoObj); err != nil {
			return nil, err
		}
		return splitIgnorePaths(ignoreInfoObj.Ignore), nil
	}
	return nil, nil
}

// appDefFromCompiled builds app def from compiled result
func appDefFromCompiled(compileRes *compileResult) (appdef.IAppDef, error) {
	var errs []error

	builder := appdef.New()
	if err := parser.BuildAppDefs(compileRes.appSchemaAST, builder); err != nil {
		errs = append(errs, err)
	}

	appDef, err := builder.Build()
	if err != nil {
		errs = append(errs, err)
	}
	return appDef, errors.Join(errs...)
}

// appDefFromBaselineDir builds app def from baseline dir
func appDefFromBaselineDir(baselineDir string) (appdef.IAppDef, error) {
	var errs []error

	// gather schema files from baseline dir
	var schemaFiles []string
	if err := filepath.Walk(baselineDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".sql" {
			schemaFiles = append(schemaFiles, path)
		}
		return nil
	}); err != nil {
		errs = append(errs, err)
	}

	// form package files structure
	pkgFiles := make(packageFiles)
	for _, schemaFile := range schemaFiles {
		dir := filepath.Dir(schemaFile)
		qpn := strings.TrimPrefix(dir, baselineDir+"/")
		pkgFiles[qpn] = append(pkgFiles[qpn], schemaFile)
	}

	// build package ASTs from schema files
	var packageASTs []*parser.PackageSchemaAST
	for qpn, files := range pkgFiles {
		// build file ASTs
		var fileASTs []*parser.FileSchemaAST
		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				errs = append(errs, err)
			}
			fileName := filepath.Base(file)

			fileAST, err := parser.ParseFile(fileName, string(content))
			if err != nil {
				errs = append(errs, err)
			}
			fileASTs = append(fileASTs, fileAST)
		}

		// build package AST
		packageAST, err := parser.BuildPackageSchema(qpn, fileASTs)
		if err != nil {
			errs = append(errs, err)
		}
		// add package AST to list
		packageASTs = append(packageASTs, packageAST)
	}

	// build app AST
	appAST, err := parser.BuildAppSchema(packageASTs)
	if err != nil {
		errs = append(errs, err)
	}
	// build app def from app AST
	builder := appdef.New()
	if err := parser.BuildAppDefs(appAST, builder); err != nil {
		errs = append(errs, err)
	}
	appDef, err := builder.Build()
	if err != nil {
		errs = append(errs, err)
	}

	return appDef, errors.Join(errs...)
}

// splitIgnorePaths splits list of ignore paths into list of path parts
func splitIgnorePaths(ignores []string) (res [][]string) {
	res = make([][]string, len(ignores))
	for i, ignore := range ignores {
		res[i] = strings.Split(ignore, "/")
	}
	return
}
