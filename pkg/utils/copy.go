/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"io"
	"os"
	"path/filepath"
)

func CopyDir(src, dst string) error {
	srcinfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}
	
	dirEntries, err := os.ReadDir(src)
	if err != nil {
		// notest
		return err
	}

	for _, dirEntry := range dirEntries {
		srcFilePath := filepath.Join(src, dirEntry.Name())
		dstFilePath := filepath.Join(dst, dirEntry.Name())
		if dirEntry.IsDir() {
			err = CopyDir(srcFilePath, dstFilePath)
		} else {
			err = CopyFile(srcFilePath, dstFilePath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func CopyFile(src, dst string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()

	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstF.Close()

	if _, err = io.Copy(dstF, srcF); err != nil {
		// notest
		return err
	}

	srcinfo, err := os.Stat(src)
	if err != nil {
		// notest
		return err
	}

	return os.Chmod(dst, srcinfo.Mode())
}
