/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"errors"
	"fmt"
	"github.com/voedger/voedger/pkg/compile"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/exec"
	"github.com/voedger/voedger/pkg/goutils/logger"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

func newBuildCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [-C] [-o <archive-name>]",
		Short: "build",
		RunE: func(cmd *cobra.Command, args []string) error {
			compileRes, err := compile.CompileNoDummyApp(params.Dir)
			if err := checkAppSchemaNotFoundErr(err); err != nil {
				return err
			}
			if err := checkCompileResult(compileRes); err != nil {
				return err
			}
			return build(compileRes, params)
		},
	}
	cmd.SilenceErrors = true
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
	cmd.Flags().StringVarP(&params.Output, "output", "o", "", "output archive name")
	return cmd
}

func checkAppSchemaNotFoundErr(err error) error {
	if err != nil {
		logger.Error(err)
		if errors.Is(err, compile.ErrAppSchemaNotFound) {
			return errors.New("failed to build, app schema not found")
		}
	}
	return nil
}

func checkCompileResult(compileRes *compile.Result) error {
	switch {
	case compileRes == nil:
		return errors.New("failed to compile, check schemas")
	case len(compileRes.NotFoundDeps) > 0:
		return errors.New("failed to compile, missing dependencies. Run 'vpm tidy'")
	default:
		return nil
	}
}

func build(compileRes *compile.Result, params *vpmParams) error {
	// directory to save the build info: vsql files, wasm files
	buildInfoDir := filepath.Join(params.Dir, buildDirName)
	// create build info directory along with vsql and wasm files
	if err := buildDir(compileRes.PkgFiles, buildInfoDir); err != nil {
		return err
	}
	// set the path to the output archive, e.g. app.var
	archiveName := params.Output
	if archiveName == "" {
		archiveName = filepath.Base(params.Dir)
	}
	if !strings.HasSuffix(archiveName, ".var") {
		archiveName += ".var"
	}
	archivePath := filepath.Join(params.Dir, archiveName)

	// zip build info directory along with vsql and wasm files
	return coreutils.Zip(archivePath, buildInfoDir)
}

// buildDir creates a directory structure with vsql and wasm files
func buildDir(pkgFiles packageFiles, baselineDir string) error {
	for qpn, files := range pkgFiles {
		dir := filepath.Join(baselineDir, qpn)
		if err := os.MkdirAll(dir, coreutils.FileMode_rwxrwxrwx); err != nil {
			return err
		}

		for _, file := range files {
			// copy vsql files
			base := filepath.Base(file)
			fileNameExtensionless := base[:len(base)-len(filepath.Ext(base))]

			filePath := filepath.Join(dir, fileNameExtensionless+".vsql")

			fileContent, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filePath, fileContent, coreutils.FileMode_rw_rw_rw_); err != nil {
				return err
			}

			if err := copyFile(file, filePath); err != nil {
				return fmt.Errorf(errFmtCopyFile, file, err)
			}

			// check if packages_gen.go file exists, if it does then build the package using tinygo and add the resulting wasm file
			fileDir := filepath.Dir(file)
			exists, err := checkPackageGenFileExists(fileDir)
			if err != nil {
				return err
			}
			if exists {
				wasmFilePath, err := execTinyGoBuild(filepath.Join(fileDir, compile.PkgDirName))
				if err != nil {
					return err
				}
				if err := copyFile(wasmFilePath, filepath.Join(dir, filepath.Base(wasmFilePath))); err != nil {
					return fmt.Errorf(errFmtCopyFile, wasmFilePath, err)
				}
			}

		}
	}
	return nil
}

// execTinyGoBuild builds the project using tinygo and returns the path to the resulting wasm file
func execTinyGoBuild(dir string) (wasmFilePath string, err error) {
	var stdout io.Writer

	folderName := filepath.Base(dir)
	if logger.IsVerbose() {
		stdout = os.Stdout
	}
	wasmFileName := folderName + ".wasm"
	if err := new(exec.PipedExec).Command("tinygo", "build", "--no-debug", "-o", wasmFileName, "-scheduler=none", "-opt=2", "-gc=leaking", "-target=wasi", ".").WorkingDir(dir).Run(stdout, os.Stderr); err != nil {
		return "", err
	}
	return filepath.Join(dir, wasmFileName), nil
}
