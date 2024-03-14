/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	//"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestCompileBasicUsage(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()

	require.NoError(err)

	testCases := []struct {
		name string
		dir  string
	}{
		{
			name: "simple schema with no imports",
			dir:  filepath.Join(wd, "test", "myapp", "mypkg1"),
		},
		{
			name: "schema importing a local package",
			dir:  filepath.Join(wd, "test", "myapp", "mypkg2"),
		},
		{
			name: "app schema importing voedger package",
			dir:  filepath.Join(wd, "test", "myapp", "app"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//err := os.Chdir(tc.dir)
			//require.NoError(err)

			err = execRootCmd([]string{"vpm", "compile", "-C", tc.dir}, "1.0.0")
			require.NoError(err)
		})
	}
}

func TestBaselineBasicUsage(t *testing.T) {
	logger.SetLogLevel(logger.LogLevelVerbose)
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)

	tempTargetDir := t.TempDir()
	baselineDirName := "baseline_schemas"
	testCases := []struct {
		name                  string
		dir                   string
		expectedBaselineFiles []string
	}{
		{
			name: "simple schema with no imports",
			dir:  filepath.Join(wd, "test", "myapp", "mypkg1"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg1", "schema1.sql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name: "schema importing a local package",
			dir:  filepath.Join(wd, "test", "myapp", "mypkg2"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg1", "schema1.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg2", "schema2.sql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name: "application schema using both local package and voedger",
			dir:  filepath.Join(wd, "test", "myapp", "app"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg1", "schema1.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg2", "schema2.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "app", "app.sql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//err := os.Chdir(tc.dir)
			//require.NoError(err)

			err = os.RemoveAll(tempTargetDir)
			require.NoError(err)

			baselineDir := filepath.Join(tempTargetDir, baselineDirName)
			err = execRootCmd([]string{"vpm", "baseline", "-C", tc.dir, baselineDir}, "1.0.0")
			require.NoError(err)

			var actualFilePaths []string
			err = filepath.Walk(tempTargetDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					actualFilePaths = append(actualFilePaths, path)
				}
				return nil
			})
			require.NoError(err)

			require.Equal(len(tc.expectedBaselineFiles), len(actualFilePaths))
			for _, actualFilePath := range actualFilePaths {
				require.Contains(tc.expectedBaselineFiles, actualFilePath)
			}
		})
	}
}

func TestCompatBasicUsage(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)

	tempDir := t.TempDir()
	workDir := filepath.Join(wd, "test", "myapp", "app")
	baselineDir := filepath.Join(tempDir, "test", "baseline_myapp")
	err = execRootCmd([]string{"vpm", "baseline", baselineDir, "--change-dir", workDir}, "1.0.0")
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "compat", "-C", workDir, baselineDir}, "1.0.0")
	require.NoError(err)
}

func TestCompatErrors(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)

	tempDir := t.TempDir()
	workDir := filepath.Join(wd, "test", "myapp", "app")
	baselineDir := filepath.Join(tempDir, "test", "baseline_myapp")
	err = execRootCmd([]string{"vpm", "baseline", "-C", workDir, baselineDir}, "1.0.0")
	require.NoError(err)

	workDir = filepath.Join(wd, "test", "myapp_incompatible", "app")
	err = execRootCmd([]string{"vpm", "compat", "--ignore", filepath.Join(workDir, "ignores.yml"), "--change-dir", workDir, baselineDir}, "1.0.0")
	require.Error(err)
	errs := coreutils.SplitErrors(err)

	expectedErrs := []string{
		"OrderChanged: AppDef/Types/mypkg2.MyTable2/Fields/myfield3",
		"OrderChanged: AppDef/Types/mypkg2.MyTable2/Fields/myfield2",
	}
	require.Equal(len(expectedErrs), len(errs))

	for _, err := range errs {
		require.Contains(expectedErrs, err.Error())
	}
}

func TestCompileErrors(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)

	testCases := []struct {
		name                 string
		dir                  string
		expectedErrPositions []string
	}{
		{
			name: "schema1.sql - syntax errors",
			dir:  filepath.Join(wd, "test", "myapperr", "mypkg1"),
			expectedErrPositions: []string{
				"schema1.sql:7:28",
			},
		},
		{
			name: "schema2.sql - syntax errors",
			dir:  filepath.Join(wd, "test", "myapperr", "mypkg2"),
			expectedErrPositions: []string{
				"schema2.sql:7:13",
			},
		},
		{
			name: "schema4.sql - package local name redeclared",
			dir:  filepath.Join(wd, "test", "myapperr", "app"),
			expectedErrPositions: []string{
				"schema4.sql:5:1: local package name reg was redeclared as registry",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = execRootCmd([]string{"vpm", "compile", "-C", tc.dir}, "1.0.0")
			require.Error(err)
			errMsg := err.Error()
			for _, expectedErrPosition := range tc.expectedErrPositions {
				require.Contains(errMsg, expectedErrPosition)
			}
			fmt.Println(err.Error())
		})
	}
}

func TestPkgRegistryCompile(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	pkgDirLocalPath := wd[:strings.LastIndex(wd, filepath.FromSlash("/cmd/vpm"))] + filepath.FromSlash("/pkg")

	require.NoError(err)
	defer func() {
		_ = os.Chdir(wd)
	}()

	err = os.Chdir(pkgDirLocalPath)
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "compile", "-C", "registry"}, "1.0.0")
	require.NoError(err)
}

func TestGenOrmBasicUsage(t *testing.T) {
	require := require.New(t)

	// !!! Does not work in temp dir because of ref to staging version of exttinygo package in go.work file
	//tempDir := t.TempDir()
	// !!! But works in current dir
	wd, err := os.Getwd()
	require.NoError(err)

	//err = copy.Copy(filepath.Join(wd, "test", "genorm"), tempDir)
	//require.NoError(err)

	//dir := filepath.Join(tempDir, "app")
	dir := filepath.Join(wd, "test", "genorm", "app")
	headerFile := filepath.Join(dir, "header.txt")
	err = execRootCmd([]string{"vpm", "gen", "orm", "-C", dir, "--header-file", headerFile}, "1.0.0")
	require.NoError(err)

	err = new(exec.PipedExec).Command("go", "build", "-C", dir).Run(os.Stdout, os.Stderr)
	require.NoError(err)
}
