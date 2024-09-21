/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Zip(sourceDir string, zipFileName string) error {
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		// notest
		return err
	}
	exists, err := Exists(zipFileName)
	if err != nil {
		// notest
		return err
	}
	if exists {
		return fmt.Errorf("%s file exists already", zipFileName)
	}
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		// notest
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.WalkDir(sourceDir, func(pathToZip string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			// notest
			return err
		}

		if pathToZip == absSourceDir {
			return nil
		}

		if pathToZip == zipFile.Name() {
			// skip the zip file itself if it is placed within the dir we're zipping
			return nil
		}

		info, err := dirEntry.Info()
		if err != nil {
			// notest
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			// notest
			return err
		}

		relPath, err := filepath.Rel(absSourceDir, pathToZip)
		if err != nil {
			// notest
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			// notest
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileToZip, err := os.Open(pathToZip)
		if err != nil {
			// notest
			return err
		}
		defer fileToZip.Close()

		_, err = io.Copy(writer, fileToZip)
		return err
	})
}

func Unzip(zipFileName, destDir string) error {
	reader, err := zip.OpenReader(zipFileName)
	if err != nil {
		// notest
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		zippedFile, err := file.Open()
		if err != nil {
			// notest
			return err
		}

		destFilePath := filepath.Join(destDir, file.Name) // #nosec G305
		if err := os.MkdirAll(filepath.Dir(destFilePath), FileMode_rwxrwxrwx); err != nil {
			// notest
			return err
		}

		destFile, err := os.Create(destFilePath)
		if err != nil {
			// notest
			return err
		}

		_, err = io.Copy(destFile, zippedFile) // #nosec G110

		zippedFile.Close()
		destFile.Close()

		if err != nil {
			// notest
			return err
		}
	}
	return nil
}
