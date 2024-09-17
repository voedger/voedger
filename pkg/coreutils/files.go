/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
)

// copies the specified dir from the provided FS to disk to path specified by dst
// use "." src to copy the entire srcFS content
func CopyDirFS(srcFS IReadFS, src, dst string, optFuncs ...CopyOpt) error {
	opts := &copyOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	return copyDirFSOpts(srcFS, src, dst, opts)
}

// copies the specified file from the provided FS to disk to path specified by dstDir
func CopyFileFS(srcFS fs.FS, srcFileName, dstDir string, optFuncs ...CopyOpt) error {
	opts := &copyOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	return copyFileFSOpts(srcFS, srcFileName, dstDir, opts)
}

func CopyFile(src, dstDir string, optFuncs ...CopyOpt) error {
	dir, file := filepath.Split(filepath.Clean(src))
	if len(dir) == 0 {
		dir = "."
	}
	return CopyFileFS(os.DirFS(dir), file, dstDir, optFuncs...)
}

func CopyDir(src, dst string, optFuncs ...CopyOpt) error {
	readDirFS := os.DirFS(src).(IReadFS)
	return CopyDirFS(readDirFS, ".", dst, optFuncs...)
}

func copyDirFSOpts(srcFS IReadFS, src, dst string, opts *copyOpts) error {
	srcinfo, err := fs.Stat(srcFS, src)
	if err != nil {
		// notest
		return err
	}

	// TODO: src is "." -> srcinfo.Mode() is weak -> permission deined on create dst within temp dir created with more strong FileMode
	_ = srcinfo
	if err = os.MkdirAll(dst, FileMode_rwxrwxrwx); err != nil {
		return err
	}

	dirEntries, err := srcFS.ReadDir(src)
	if err != nil {
		// notest
		return err
	}

	for _, dirEntry := range dirEntries {
		srcFilePath := path.Join(src, dirEntry.Name()) // '/' separator must be used for fs.FS,
		if dirEntry.IsDir() {
			dstFilePath := filepath.Join(dst, dirEntry.Name())
			err = copyDirFSOpts(srcFS, srcFilePath, dstFilePath, opts)
		} else {
			err = copyFileFSOpts(srcFS, srcFilePath, dst, opts)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFileFSOpts(srcFS fs.FS, srcFileName, dstDir string, opts *copyOpts) error {
	if len(opts.files) > 0 && !slices.Contains(opts.files, srcFileName) {
		return nil
	}
	srcF, err := srcFS.Open(srcFileName)
	if err != nil {
		return err
	}
	defer srcF.Close()

	targetFileName := filepath.Base(srcFileName)
	if len(opts.targetFileName) > 0 {
		targetFileName = opts.targetFileName
	}
	dstFileNameWithPath := filepath.Join(dstDir, targetFileName)
	existsDstFile, err := Exists(dstFileNameWithPath)
	if err != nil {
		// notest
		return err
	}
	if existsDstFile {
		if opts.skipExisting {
			return nil
		}
		return fmt.Errorf("file %s already exists: %w", dstDir, os.ErrExist)
	}

	existsDstDir, err := Exists(dstDir)
	if err != nil {
		// notest
		return err
	}
	if !existsDstDir {
		if err := os.MkdirAll(dstDir, FileMode_rwxrwxrwx); err != nil {
			// notest
			return err
		}
	}
	dstF, err := os.Create(dstFileNameWithPath)
	if err != nil {
		return err
	}
	defer dstF.Close()

	if _, err = io.Copy(dstF, srcF); err != nil {
		// notest
		return err
	}

	if err = dstF.Sync(); err != nil {
		// notest
		return err
	}

	if opts.fm > 0 {
		return os.Chmod(dstFileNameWithPath, opts.fm)
	}

	srcinfo, err := fs.Stat(srcFS, srcFileName)
	if err != nil {
		// notest
		return err
	}

	return os.Chmod(dstFileNameWithPath, srcinfo.Mode())
}

func Exists(filePath string) (exists bool, err error) {
	if _, err = os.Stat(filePath); err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		err = nil
	}
	return false, err
}

type copyOpts struct {
	fm             fs.FileMode
	skipExisting   bool
	files          []string
	targetFileName string
}

type CopyOpt func(co *copyOpts)

func WithFileMode(fm fs.FileMode) CopyOpt {
	return func(co *copyOpts) {
		co.fm = fm
	}
}

func WithSkipExisting() CopyOpt {
	return func(co *copyOpts) {
		co.skipExisting = true
	}
}

func WithNewName(fileName string) CopyOpt {
	return func(co *copyOpts) {
		co.targetFileName = fileName
	}
}

func WithFilterFilesWithRelativePaths(filesWithRelativePaths []string) CopyOpt {
	return func(co *copyOpts) {
		co.files = filesWithRelativePaths
	}
}
