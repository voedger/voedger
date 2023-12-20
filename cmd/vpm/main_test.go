/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

//go:embed test/myapp/*
var testMyAppFS embed.FS

//go:embed test/myapp_incompatible/*
var testMyAppIncompatibleFS embed.FS

//go:embed test/myapperr/*
var testMyAppErrFS embed.FS

func TestCompileBasicUsage(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)
	defer func() {
		_ = os.Chdir(wd)
	}()

	tempDir := t.TempDir()
	err = copyContents(testMyAppFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	testCases := []struct {
		name string
		dir  string
	}{
		{
			name: "simple schema with no imports",
			dir:  filepath.Join(tempDir, "test", "myapp", "mypkg1"),
		},
		{
			name: "schema importing a local package",
			dir:  filepath.Join(tempDir, "test", "myapp", "mypkg2"),
		},
		{
			name: "schema importing voedger package",
			dir:  filepath.Join(tempDir, "test", "myapp", "mypkg3"),
		},
		{
			name: "application schema using both local package and voedger",
			dir:  filepath.Join(tempDir, "test", "myapp"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.Chdir(tc.dir)
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "compile", "-C", tc.dir}, "1.0.0")
			require.NoError(err)
		})
	}
}

func TestBaselineBasicUsage(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)
	defer func() {
		_ = os.Chdir(wd)
	}()

	tempTargetDir := t.TempDir()
	tempDir := t.TempDir()
	err = copyContents(testMyAppFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	baselineDirName := "baseline_schemas"
	testCases := []struct {
		name                  string
		workingDir            string
		expectedBaselineFiles []string
	}{
		{
			name:       "simple schema with no imports",
			workingDir: filepath.Join(tempDir, "test", "myapp", "mypkg1"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "mypkg1", "schema1.sql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name:       "schema importing a local package",
			workingDir: filepath.Join(tempDir, "test", "myapp", "mypkg2"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "mypkg1", "schema1.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "mypkg2", "schema2.sql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name:       "schema importing voedger package",
			workingDir: filepath.Join(tempDir, "test", "myapp", "mypkg3"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "mypkg3", "schema3.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "github.com", "voedger", "voedger", "pkg", "registry", "schemas.sql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name:       "application schema using both local package and voedger",
			workingDir: filepath.Join(tempDir, "test", "myapp"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "github.com", "voedger", "voedger", "pkg", "registry", "schemas.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "mypkg1", "schema1.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "mypkg2", "schema2.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "mypkg3", "schema3.sql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "server.com", "account", "repo", "myapp.sql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.Chdir(tc.workingDir)
			require.NoError(err)

			err = os.RemoveAll(tempTargetDir)
			require.NoError(err)

			baselineDir := filepath.Join(tempTargetDir, baselineDirName)
			err = execRootCmd([]string{"vpm", "baseline", "-C", tc.workingDir, baselineDir}, "1.0.0")
			require.NoError(err)

			var actualFilePaths []string
			err = filepath.Walk(tempTargetDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
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
	defer func() {
		_ = os.Chdir(wd)
	}()

	tempDir := t.TempDir()
	err = copyContents(testMyAppFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	workDir := filepath.Join(tempDir, "test", "myapp")
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
	defer func() {
		_ = os.Chdir(wd)
	}()

	tempDir := t.TempDir()
	err = copyContents(testMyAppFS, tempDir)
	require.NoError(err)

	err = copyContents(testMyAppIncompatibleFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	workDir := filepath.Join(tempDir, "test", "myapp")
	baselineDir := filepath.Join(tempDir, "test", "baseline_myapp")
	err = execRootCmd([]string{"vpm", "baseline", "-C", workDir, baselineDir}, "1.0.0")
	require.NoError(err)

	workDir = filepath.Join(tempDir, "test", "myapp_incompatible")
	err = execRootCmd([]string{"vpm", "compat", "--ignore", filepath.Join(workDir, "ignores.yml"), "--change-dir", workDir, baselineDir}, "1.0.0")
	require.Error(err)
	errs := coreutils.SplitErrors(err)

	expectedErrs := []string{
		"OrderChanged: AppDef/Types/mypkg2.MyTable/Fields/myfield3",
		"OrderChanged: AppDef/Types/mypkg2.MyTable/Fields/myfield2",
		"NodeRemoved: AppDef/Types/mypkg3.MyTable3/Fields/MyField",
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
	defer func() {
		_ = os.Chdir(wd)
	}()

	tempDir := t.TempDir()
	err = copyContents(testMyAppErrFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	testCases := []struct {
		name                 string
		dir                  string
		expectedErrPositions []string
	}{
		{
			name: "package schema - syntax errors",
			dir:  filepath.Join(tempDir, "test", "myapperr", "mypkg1"),
			expectedErrPositions: []string{
				"schema1.sql:7:33",
			},
		},
		{
			name: "application schema - syntax errors",
			dir:  filepath.Join(tempDir, "test", "myapperr", "mypkg2"),
			expectedErrPositions: []string{
				"schema2.sql:7:5",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.Chdir(tc.dir)
			require.NoError(err)

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

func copyContents(src embed.FS, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	err := fs.WalkDir(src, ".", func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate the destination path
		destPath := filepath.Join(dest, entryPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		fileName := filepath.Base(entryPath)
		// we can't have embed.FS of the dir with go.mod file inside,
		// that's why we name it test.go.mod and rename it back
		if fileName == "test.go.mod" {
			destPath = filepath.Join(filepath.Dir(destPath), "go.mod")
		}
		srcFile, err := src.Open(entryPath)
		if err != nil {
			return err
		}
		defer func() {
			_ = srcFile.Close()
		}()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer func() {
			_ = destFile.Close()
		}()

		_, err = io.Copy(destFile, srcFile)
		return err
	})

	return err
}
