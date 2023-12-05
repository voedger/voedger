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
	"github.com/untillpro/goutils/exec"

	"github.com/voedger/voedger/cmd/vpm/internal/dm"
)

//go:embed test/myapp/*
var testMyAppFS embed.FS

//go:embed test/myapperr/*
var testMyAppErrFS embed.FS

func TestBasicUsage(t *testing.T) {
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
			dir:  fmt.Sprintf("%s/test/myapperr/mypkgerr", tempDir),
			expectedErrPositions: []string{
				"schema1.sql:4:33",
			},
		},
		{
			name: "application schema - syntax errors",
			dir:  fmt.Sprintf("%s/test/myapperr", tempDir),
			expectedErrPositions: []string{
				"schema1.sql:4:33",
				"schema2.sql:4:5",
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

func TestMissedDependency(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)
	defer func() {
		_ = os.Chdir(wd)
	}()

	tempDir := t.TempDir()

	err = copyContents(testMyAppFS, tempDir)
	require.NoError(err)

	testDir := fmt.Sprintf("%s/test/myapp", tempDir)
	err = os.Chdir(tempDir)
	require.NoError(err)

	t.Run("run normal", func(t *testing.T) {
		err := os.Chdir(testDir)
		require.NoError(err)

		err = execRootCmd([]string{"vpm", "compile", fmt.Sprintf(" -C %s", testDir)}, "1.0.0")
		require.NoError(err)
	})

	t.Run("clean dependency cache", func(t *testing.T) {
		err := os.Chdir(testDir)
		require.NoError(err)

		goCacheCleanCmd := new(exec.PipedExec).Command("go", "clean", "-modcache")
		err = goCacheCleanCmd.Run(nil, nil)
		require.NoError(err)

		goDM, err := dm.NewGoBasedDependencyManager()
		require.NoError(err)

		localPath := path.Join(goDM.CachePath(), "github.com/voedger/voedger/pkg/sys")

		_, err = os.Stat(localPath)
		require.True(os.IsNotExist(err))
	})

	t.Run("rerun normal", func(t *testing.T) {
		err := os.Chdir(testDir)
		require.NoError(err)

		err = execRootCmd([]string{"vpm", "compile", fmt.Sprintf(" -C %s", testDir)}, "1.0.0")
		require.NoError(err)
	})

	t.Run("check dependency integrity after", func(t *testing.T) {
		err := os.Chdir(testDir)
		require.NoError(err)

		goDM, err := dm.NewGoBasedDependencyManager()
		require.NoError(err)

		localPath, err := goDM.LocalPath("github.com/voedger/voedger/pkg/sys")
		require.NoError(err)

		_, err = os.Stat(localPath)
		require.False(os.IsNotExist(err))
	})
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
