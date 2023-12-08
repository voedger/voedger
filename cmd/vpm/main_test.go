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
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test/myapp/*
var testMyAppFS embed.FS

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
			dir:  path.Join(tempDir, "test/myapp/mypkg1"),
		},
		{
			name: "schema importing a local package",
			dir:  path.Join(tempDir, "test/myapp/mypkg2"),
		},
		{
			name: "schema importing voedger package",
			dir:  path.Join(tempDir, "test/myapp/mypkg3"),
		},
		{
			name: "application schema using both local package and voedger",
			dir:  path.Join(tempDir, "test/myapp"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.Chdir(tc.dir)
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "compile", fmt.Sprintf(" -C %s", tc.dir)}, "1.0.0")
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

	testCases := []struct {
		name                  string
		workingDir            string
		expectedBaselineFiles []string
	}{
		{
			name:       "simple schema with no imports",
			workingDir: path.Join(tempDir, "test/myapp/mypkg1"),
			expectedBaselineFiles: []string{
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/sys/sys.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/mypkg1/schema1.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name:       "schema importing a local package",
			workingDir: path.Join(tempDir, "test/myapp/mypkg2"),
			expectedBaselineFiles: []string{
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/sys/sys.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/mypkg1/schema1.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/mypkg2/schema2.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name:       "schema importing voedger package",
			workingDir: path.Join(tempDir, "test/myapp/mypkg3"),
			expectedBaselineFiles: []string{
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/sys/sys.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/mypkg3/schema3.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/github.com/voedger/voedger/pkg/registry/schemas.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name:       "application schema using both local package and voedger",
			workingDir: path.Join(tempDir, "test/myapp"),
			expectedBaselineFiles: []string{
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/sys/sys.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/github.com/voedger/voedger/pkg/registry/schemas.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/mypkg1/schema1.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/mypkg2/schema2.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/mypkg3/schema3.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, fmt.Sprintf("%s/%s/server.com/account/repo/myapp.sql", baselineDirName, pkgDirName)),
				path.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.Chdir(tc.workingDir)
			require.NoError(err)

			err = os.RemoveAll(path.Join(tempTargetDir, baselineDirName))
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "baseline", fmt.Sprintf(" -C %s", tc.workingDir), tempTargetDir}, "1.0.0")
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

func TestErrorsInCompile(t *testing.T) {
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
			dir:  path.Join(tempDir, "test/myapperr/mypkg1"),
			expectedErrPositions: []string{
				"schema1.sql:7:33",
			},
		},
		{
			name: "application schema - syntax errors",
			dir:  path.Join(tempDir, "test/myapperr/mypkg2"),
			expectedErrPositions: []string{
				"schema2.sql:7:5",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.Chdir(tc.dir)
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "compile", fmt.Sprintf(" -C %s", tc.dir)}, "1.0.0")
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
		destPath := path.Join(dest, entryPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		fileName := path.Base(entryPath)
		// we can't have embed.FS of the dir with go.mod file inside,
		// that's why we name it test.go.mod and rename it back
		if fileName == "test.go.mod" {
			destPath = path.Join(path.Dir(destPath), "go.mod")
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
