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

func CopyDirFS(srcFS IReadFS, src, dst string, optFuncs ...CopyOpt) error {
	opts := &copyOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	return copyDirFSOpts(srcFS, src, dst, opts)
}

func CopyFileFS(srcFS fs.FS, src, dst string, optFuncs ...CopyOpt) error {
	opts := &copyOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	return copyFileFSOpts(srcFS, src, dst, opts)
}

func CopyFile(src, dst string) error {
	return CopyFileFS(os.DirFS(filepath.Clean(src)), src, dst)
}

func CopyDir(src, dst string) error {
	readDirFS := os.DirFS(src).(IReadFS)
	return CopyDirFS(readDirFS, ".", dst)
}

func copyDirFSOpts(srcFS IReadFS, src, dst string, opts *copyOpts) error {
	srcinfo, err := fs.Stat(srcFS, src)
	if err != nil {
		// notest
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	dirEntries, err := srcFS.ReadDir(src)
	if err != nil {
		// notest
		return err
	}

	for _, dirEntry := range dirEntries {
		srcFilePath := path.Join(src, dirEntry.Name()) // '/' separator must be used for fs.FS,
		dstFilePath := filepath.Join(dst, dirEntry.Name())
		if dirEntry.IsDir() {
			err = copyDirFSOpts(srcFS, srcFilePath, dstFilePath, opts)
		} else {
			err = copyFileFSOpts(srcFS, srcFilePath, dstFilePath, opts)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFileFSOpts(srcFS fs.FS, src, dst string, opts *copyOpts) error {
	if len(opts.files) > 0 && !slices.Contains(opts.files, src) {
		return nil
	}
	srcF, err := srcFS.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()

	exists, err := Exists(dst)
	if err != nil {
		// notest
		return err
	}
	if exists {
		if opts.skipExisting {
			return nil
		}
		return fmt.Errorf("file %s already exists: %w", dst, os.ErrExist)
	}

	dstF, err := os.Create(dst)
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
		return os.Chmod(dst, opts.fm)
	}

	srcinfo, err := fs.Stat(srcFS, src)
	if err != nil {
		// notest
		return err
	}

	return os.Chmod(dst, srcinfo.Mode())
}

func Exists(filePath string) (exists bool, err error) {
	if _, err = os.Stat(filePath); err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// notest
	return false, err
}

type copyOpts struct {
	fm           fs.FileMode
	skipExisting bool
	files        []string
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

func WithFilterFiles(files []string) CopyOpt {
	return func(co *copyOpts) {
		co.files = files
	}
}
