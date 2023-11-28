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

//go:embed test/*
var testFS embed.FS

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	tempDir := t.TempDir()
	err := copyContents(testFS, tempDir)
	require.NoError(err)

	err = os.Chdir(tempDir)
	require.NoError(err)

	// TODO: ensure that we look for sql files in subdir as well
	testCases := []struct {
		dir string
	}{
		{dir: fmt.Sprintf("%s/test/mypkg1", tempDir)},
		{dir: fmt.Sprintf("%s/test/mypkg2", tempDir)},
		{dir: fmt.Sprintf("%s/test/mypkg3", tempDir)},
		{dir: fmt.Sprintf("%s/test", tempDir)},
	}

	for _, tc := range testCases {
		t.Run(tc.dir, func(t *testing.T) {
			err := os.Chdir(tc.dir)
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "compile", fmt.Sprintf(" --C=%s", tc.dir)}, "1.0.0")
			require.NoError(err)
		})
	}
}

func copyContents(src embed.FS, dest string) error {
	// Create the destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// Walk the source directory
	err := fs.WalkDir(src, ".", func(entryPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate the destination path
		destPath := path.Join(dest, entryPath)

		// If it's a directory, create it in the destination
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// If it's a file, copy it to the destination
		fileName := path.Base(entryPath)
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
