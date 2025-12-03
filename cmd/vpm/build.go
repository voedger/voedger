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
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/voedger/voedger/pkg/compile"
	"github.com/voedger/voedger/pkg/goutils/exec"
	"github.com/voedger/voedger/pkg/goutils/filesu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/zipu"
	"github.com/voedger/voedger/pkg/parser"
)

// global variables used to make version checking testable
var getTinyGoVersionFuncVariable = getTinyGoVersion

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

				return fmt.Errorf("failed to compile: %w", err)
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
	if err := os.MkdirAll(tempDir, filesu.FileMode_DefaultForDir); err != nil {
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
	return zipu.Zip(tempDirWithoutBuild, varFile)
}

// buildDir creates a directory structure with vsql and wasm files
func buildDir(pkgFiles packageFiles, buildDirPath string) error {
	wasmDirsToBuild := make([]string, 0, len(pkgFiles))
	for qpn, files := range pkgFiles {
		pkgBuildDir := filepath.Join(buildDirPath, qpn)
		if err := os.MkdirAll(pkgBuildDir, filesu.FileMode_DefaultForDir); err != nil {
			return err
		}

		for _, file := range files {
			// copy vsql files
			base := filepath.Base(file)
			fileNameExtensionless := base[:len(base)-len(filepath.Ext(base))]
			if err := filesu.CopyFile(file, pkgBuildDir, filesu.WithNewName(fileNameExtensionless+parser.VSQLExt)); err != nil {
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

			exists, err := filesu.Exists(wasmDirPath)
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
				if err := filesu.CopyFile(wasmFilePath, pkgBuildDir); err != nil {
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
	ok, err := checkTinyGoVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get tinygo version: %w", err)
	}

	if !ok {
		return "", fmt.Errorf("tinygo version is lower than %s", minimalRequiredTinyGoVersionValue)
	}

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
		"-buildmode=wasi-legacy", // see https://github.com/tinygo-org/tinygo/pull/4734
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

// getTinyGoVersion returns the version of the installed tinygo
func getTinyGoVersion() (string, error) {
	// notest
	var stdout strings.Builder

	if err := new(exec.PipedExec).Command("tinygo", "version").Run(&stdout, os.Stderr); err != nil {
		// notest
		return "", fmt.Errorf("failed to get tinygo version: %w", err)
	}

	return stdout.String(), nil
}

// checkTinyGoVersion checks if the installed tinygo version is greater than or equal to the minimal required version
func checkTinyGoVersion() (bool, error) {
	if getTinyGoVersionFuncVariable == nil {
		return false, errors.New("getTinyGoVersionFuncVariable is not set")
	}
	// Get the version of the installed tinygo
	versionOutput, err := getTinyGoVersionFuncVariable()
	if err != nil {
		return false, err
	}

	// Regex to extract version from: "tinygo version 0.33.0 darwin/arm64..."
	re := regexp.MustCompile(`tinygo version (\d+\.\d+\.?\d?)`)
	matches := re.FindStringSubmatch(versionOutput)

	if len(matches) < 2 {
		return false, fmt.Errorf("could not parse tinygo version from: %s", versionOutput)
	}

	tinyGoVersion := matches[1]

	return semver.Compare("v"+tinyGoVersion, "v"+minimalRequiredTinyGoVersionValue) >= 0, nil
}
