/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/voedger/voedger/pkg/parser"

	"github.com/google/uuid"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/compile"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/exec"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func newBuildCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "build application .var file",
		RunE: func(cmd *cobra.Command, args []string) error {
			exists, err := checkPackageGenFileExists(params.Dir)
			if err != nil {
				return err
			}
			if !exists {
				return errors.New("packages_gen.go not found. Run 'vpm init'")
			}

			compileRes, err := compile.CompileNoDummyApp(params.Dir)
			if err != nil {
				if errors.Is(err, compile.ErrAppSchemaNotFound) {
					return errors.New("failed to build, app schema not found")
				}

				return errors.New("failed to compile, check schemas")
			}

			if len(compileRes.NotFoundDeps) > 0 {
				return errors.New("failed to compile, missing dependencies. Run 'vpm tidy'")
			}

			return build(compileRes, params)
		},
	}
	cmd.Flags().StringVarP(&params.Output, "output", "o", "", "output archive name")

	return cmd
}

func build(compileRes *compile.Result, params *vpmParams) error {
	// temp directory to save the build info: vsql files, wasm files
	tempDir := filepath.Join(os.TempDir(), uuid.New().String(), buildDirName)
	if err := os.MkdirAll(tempDir, coreutils.FileMode_rwxrwxrwx); err != nil {
		return err
	}
	// create temp build info directory along with vsql and wasm files
	if err := buildDir(compileRes.PkgFiles, tempDir); err != nil {
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
	varFile := filepath.Join(params.Dir, archiveName)

	// set dir without "build" dir on the end. That need to have expected path within the archive: build/file1.txt instead of file1.txt
	tempDirWithoutBuild := filepath.Dir(tempDir)

	// zip build info directory along with vsql and wasm files
	return coreutils.Zip(tempDirWithoutBuild, varFile)
}

// buildDir creates a directory structure with vsql and wasm files
func buildDir(pkgFiles packageFiles, buildDirPath string) error {
	wasmDirsToBuild := make([]string, 0, len(pkgFiles))
	for qpn, files := range pkgFiles {
		pkgBuildDir := filepath.Join(buildDirPath, qpn)
		if err := os.MkdirAll(pkgBuildDir, coreutils.FileMode_rwxrwxrwx); err != nil {
			return err
		}

		for _, file := range files {
			// copy vsql files
			base := filepath.Base(file)
			fileNameExtensionless := base[:len(base)-len(filepath.Ext(base))]
			if err := coreutils.CopyFile(file, pkgBuildDir, coreutils.WithNewName(fileNameExtensionless+parser.VSqlExt)); err != nil {
				return fmt.Errorf(errFmtCopyFile, file, err)
			}

			// building wasm files: if wasm directory exists,
			// build wasm file and copy it to the temp build directory
			fileDir := filepath.Dir(file)
			wasmDirPath := filepath.Join(fileDir, wasmDirName)
			// build only unique wasm directories
			if slices.Contains(wasmDirsToBuild, wasmDirPath) {
				continue
			}

			exists, err := coreutils.Exists(wasmDirPath)
			if err != nil {
				return err
			}
			if exists {
				appName := filepath.Base(fileDir)
				wasmFilePath, err := execTinyGoBuild(wasmDirPath, appName)
				if err != nil {
					return err
				}
				// for controlling uniqueness of wasm directories
				wasmDirsToBuild = append(wasmDirsToBuild, wasmDirPath)
				// copy the wasm file to the build directory
				if err := coreutils.CopyFile(wasmFilePath, pkgBuildDir); err != nil {
					return fmt.Errorf(errFmtCopyFile, wasmFilePath, err)
				}
				// remove the wasm file after copying it to the build directory
				if err := os.Remove(wasmFilePath); err != nil {
					return err
				}
			}

		}
	}
	return nil
}

// execTinyGoBuild builds the project using tinygo and returns the path to the resulting wasm file
func execTinyGoBuild(dir, appName string) (wasmFilePath string, err error) {
	var stdout io.Writer
	if logger.IsVerbose() {
		stdout = os.Stdout
	}

	wasmFileName := appName + ".wasm"
	if err := new(exec.PipedExec).Command(
		"tinygo",
		"build",
		"--no-debug",
		"-o",
		wasmFileName,
		"-scheduler=none",
		"-opt=2",
		"-gc=leaking",
		"-target=wasi",
		".",
	).WorkingDir(dir).Run(stdout, os.Stderr); err != nil {
		// checking compatibility of the tinygo with go version
		if strings.Contains(err.Error(), "requires go version") {
			return "", fmt.Errorf("tinygo is incompatible with the current go version - %w", err)
		} else if strings.Contains(err.Error(), "error: unable to make temporary file: No such file or directory") {
			return "", fmt.Errorf(`"%w". Hint: on Windows try to create c:\Temp dir`, err)
		} else if strings.Contains(err.Error(), "error: could not find wasm-opt, set the WASMOPT environment variable to override") {
			return "", fmt.Errorf(`"%w". Hint: try to install binaryen from https://github.com/WebAssembly/binaryen/releases/`, err)
		}
		return "", err
	}
	return filepath.Join(dir, wasmFileName), nil
}
