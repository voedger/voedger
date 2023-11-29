package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test/myapp/*
var testMyAppFS embed.FS

//go:embed test/myapperr/*
var testMyAppErrFS embed.FS

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	tempDir := t.TempDir()
	err := copyContents(testMyAppFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	testCases := []struct {
		name string
		dir  string
	}{
		{
			name: "simple schema with no imports",
			dir:  fmt.Sprintf("%s/test/myapp/mypkg1", tempDir),
		},
		{
			name: "schema importing a local package",
			dir:  fmt.Sprintf("%s/test/myapp/mypkg2", tempDir),
		},
		{
			name: "schema importing voedger package",
			dir:  fmt.Sprintf("%s/test/myapp/mypkg3", tempDir),
		},
		{
			name: "application schema using both local package and voedger",
			dir:  fmt.Sprintf("%s/test/myapp", tempDir),
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

func TestErrorsInCompile(t *testing.T) {
	require := require.New(t)

	tempDir := t.TempDir()
	err := copyContents(testMyAppErrFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	testCases := []struct {
		name string
		dir  string
	}{
		{
			name: "simple schema",
			dir:  fmt.Sprintf("%s/test/myapperr/mypkgerr", tempDir),
		},
		{
			name: "application schema",
			dir:  fmt.Sprintf("%s/test/myapperr", tempDir),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := os.Chdir(tc.dir)
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "compile", fmt.Sprintf(" -C %s", tc.dir)}, "1.0.0")
			require.Error(err)
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
