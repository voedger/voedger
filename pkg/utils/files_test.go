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

	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	require := require.New(t)
	tempDirSrc := t.TempDir()
	tempDirDst := t.TempDir()
	dir1 := filepath.Join(tempDirSrc, "dir1")
	dir2 := filepath.Join(tempDirSrc, "dir2")
	dir1dir3 := filepath.Join(dir1, "dir3")
	require.NoError(os.MkdirAll(dir1, 0777))
	require.NoError(os.MkdirAll(dir2, 0777))
	require.NoError(os.MkdirAll(dir1dir3, 0777))
	file0 := filepath.Join(tempDirSrc, "file0.txt")
	file1 := filepath.Join(dir1, "file1.txt")
	file2 := filepath.Join(dir2, "file2.txt")
	file3 := filepath.Join(dir1dir3, "file3.txt")

	require.NoError(os.WriteFile(file0, []byte("file0 content"), 0777))
	require.NoError(os.WriteFile(file1, []byte("file1 content"), 0777))
	require.NoError(os.WriteFile(file2, []byte("file2 content"), 0777))
	require.NoError(os.WriteFile(file3, []byte("file3 content"), 0777))

	require.NoError(CopyDir(tempDirSrc, tempDirDst))
	file0 = strings.ReplaceAll(file0, tempDirSrc, tempDirDst)
	file1 = strings.ReplaceAll(file1, tempDirSrc, tempDirDst)
	file2 = strings.ReplaceAll(file2, tempDirSrc, tempDirDst)
	file3 = strings.ReplaceAll(file3, tempDirSrc, tempDirDst)

	file0ActualContent, err := os.ReadFile(file0)
	require.NoError(err)
	file1ActualContent, err := os.ReadFile(file1)
	require.NoError(err)
	file2ActualContent, err := os.ReadFile(file2)
	require.NoError(err)
	file3ActualContent, err := os.ReadFile(file3)
	require.NoError(err)

	require.Equal("file0 content", string(file0ActualContent))
	require.Equal("file1 content", string(file1ActualContent))
	require.Equal("file2 content", string(file2ActualContent))
	require.Equal("file3 content", string(file3ActualContent))
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
