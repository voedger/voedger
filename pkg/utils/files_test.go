/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCopy_BasicUsage(t *testing.T) {
	require := require.New(t)
	tempDirSrc := t.TempDir()
	dir1 := filepath.Join(tempDirSrc, "dir1")
	dir2 := filepath.Join(tempDirSrc, "dir2")
	dir1dir3 := filepath.Join(dir1, "dir3")
	require.NoError(os.MkdirAll(dir1, FileMode_rwxrwxrwx))
	require.NoError(os.MkdirAll(dir2, FileMode_rwxrwxrwx))
	require.NoError(os.MkdirAll(dir1dir3, FileMode_rwxrwxrwx))
	file0 := filepath.Join(tempDirSrc, "file0.txt")
	file1 := filepath.Join(dir1, "file1.txt")
	file2 := filepath.Join(dir2, "file2.txt")
	file3 := filepath.Join(dir1dir3, "file3.txt")

	require.NoError(os.WriteFile(file0, []byte("file0 content"), FileMode_rw_rw_rw_))
	require.NoError(os.WriteFile(file1, []byte("file1 content"), FileMode_rw_rw_rw_))
	require.NoError(os.WriteFile(file2, []byte("file2 content"), FileMode_rw_rw_rw_))
	require.NoError(os.WriteFile(file3, []byte("file3 content"), FileMode_rw_rw_rw_))

	t.Run("CopyDir", func(t *testing.T) {
		tempDirDst := t.TempDir()

		require.NoError(CopyDir(tempDirSrc, tempDirDst))

		file0Dst := strings.ReplaceAll(file0, tempDirSrc, tempDirDst)
		file1Dst := strings.ReplaceAll(file1, tempDirSrc, tempDirDst)
		file2Dst := strings.ReplaceAll(file2, tempDirSrc, tempDirDst)
		file3Dst := strings.ReplaceAll(file3, tempDirSrc, tempDirDst)

		file0ActualContent, err := os.ReadFile(file0Dst)
		require.NoError(err)
		file1ActualContent, err := os.ReadFile(file1Dst)
		require.NoError(err)
		file2ActualContent, err := os.ReadFile(file2Dst)
		require.NoError(err)
		file3ActualContent, err := os.ReadFile(file3Dst)
		require.NoError(err)

		require.Equal("file0 content", string(file0ActualContent))
		require.Equal("file1 content", string(file1ActualContent))
		require.Equal("file2 content", string(file2ActualContent))
		require.Equal("file3 content", string(file3ActualContent))
	})

	t.Run("CopyFile", func(t *testing.T) {
		tempDirDst := t.TempDir()
		require.NoError(CopyFile(file1, tempDirDst))
		file1Dst := filepath.Join(tempDirDst, filepath.Base(file1))
		file1ActualContent, err := os.ReadFile(file1Dst)
		require.NoError(err)
		require.Equal("file1 content", string(file1ActualContent))
	})

	t.Run("CopyFile to unexisting dir -> create target dir", func(t *testing.T) {
		tempDirDst := t.TempDir()
		require.NoError(os.RemoveAll(tempDirDst))
		require.NoError(CopyFile(file1, tempDirDst))
		file1Dst := filepath.Join(tempDirDst, filepath.Base(file1))
		file1ActualContent, err := os.ReadFile(file1Dst)
		require.NoError(err)
		require.Equal("file1 content", string(file1ActualContent))
	})

	t.Run("CopyFile src file without path", func(t *testing.T) {
		tempDirDst := t.TempDir()
		initialWD, err := os.Getwd()
		require.NoError(err)
		file1SrcPath, file1SrcName := filepath.Split(file1)
		require.NoError(os.Chdir(file1SrcPath))
		defer func() {
			require.NoError(os.Chdir(initialWD))
		}()

		require.NoError(CopyFile(file1SrcName, tempDirDst))
		file1Dst := filepath.Join(tempDirDst, file1SrcName)
		file1ActualContent, err := os.ReadFile(file1Dst)
		require.NoError(err)
		require.Equal("file1 content", string(file1ActualContent))
	})

	//TODO: test copy to the dir that is not exists yet, test specifying the src file without path
}

func TestCopyErrors(t *testing.T) {
	require := require.New(t)
	tempDirSrc := t.TempDir()
	tempDirDst := t.TempDir()
	unexisingDir := filepath.Join(tempDirSrc, "unexisting")

	require.Error(CopyDir("", ""))
	require.Error(CopyDir("", tempDirDst))
	require.Error(CopyDir(unexisingDir, ""))
	require.Error(CopyDir(unexisingDir, ""))
	require.Error(CopyFile("", ""))
	require.Error(CopyFile("", tempDirDst))
	require.Error(CopyFile(unexisingDir, ""))
	require.Error(CopyFile(unexisingDir, ""))
}

func TestCopyOptions(t *testing.T) {
	require := require.New(t)
	tempDirSrc := t.TempDir()
	tempDirDst := t.TempDir()
	dir1 := filepath.Join(tempDirSrc, "dir1")
	dir2 := filepath.Join(tempDirSrc, "dir2")
	dir1dir3 := filepath.Join(dir1, "dir3")
	require.NoError(os.MkdirAll(dir1, FileMode_rwxrwxrwx))
	require.NoError(os.MkdirAll(dir2, FileMode_rwxrwxrwx))
	require.NoError(os.MkdirAll(dir1dir3, FileMode_rwxrwxrwx))
	file0 := filepath.Join(tempDirSrc, "file0.txt")
	file1 := filepath.Join(dir1, "file1.txt")
	file2 := filepath.Join(dir2, "file2.txt")
	file3 := filepath.Join(dir1dir3, "file3.txt")

	require.NoError(os.WriteFile(file0, []byte("file0 content"), FileMode_rw_rw_rw_))
	require.NoError(os.WriteFile(file1, []byte("file1 content"), FileMode_rw_rw_rw_))
	require.NoError(os.WriteFile(file2, []byte("file2 content"), FileMode_rw_rw_rw_))
	require.NoError(os.WriteFile(file3, []byte("file3 content"), FileMode_rw_rw_rw_))

	file2 = strings.ReplaceAll(file2, tempDirSrc, tempDirDst)
	dir2dst := filepath.Join(tempDirDst, "dir2")
	require.NoError(os.MkdirAll(dir2dst, FileMode_rwxrwxrwx))
	require.NoError(os.WriteFile(file2, []byte("file2 existing content"), FileMode_rw_rw_rw_))

	err := CopyDir(tempDirSrc, tempDirDst, WithFilterFilesWithRelativePaths(
		[]string{"dir1/file1.txt", "dir2/file2.txt"}), WithSkipExisting())
	require.NoError(err)

	file0 = strings.ReplaceAll(file0, tempDirSrc, tempDirDst)
	file1 = strings.ReplaceAll(file1, tempDirSrc, tempDirDst)
	file2 = strings.ReplaceAll(file2, tempDirSrc, tempDirDst)
	file3 = strings.ReplaceAll(file3, tempDirSrc, tempDirDst)

	exists, err := Exists(file0)
	require.NoError(err)
	require.False(exists)
	file1ActualContent, err := os.ReadFile(file1)
	require.NoError(err)
	file2ActualContent, err := os.ReadFile(file2)
	require.NoError(err)
	exists, err = Exists(file3)
	require.NoError(err)
	require.False(exists)

	require.Equal("file1 content", string(file1ActualContent))
	require.Equal("file2 existing content", string(file2ActualContent))
}

func TestExists(t *testing.T) {
	require := require.New(t)
	t.Run("file", func(t *testing.T) {
		exists, err := Exists("files_test.go")
		require.NoError(err)
		require.True(exists)

		exists, err = Exists(uuid.New().String())
		require.NoError(err)
		require.False(exists)
	})

	t.Run("dir", func(t *testing.T) {
		exists, err := Exists("testwsdesc")
		require.NoError(err)
		require.True(exists)
	})
}
