/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
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

func Zip(zipFilePath string, objectToZip any) error {
	switch t := objectToZip.(type) {
	case string:
		fileInfo, err := os.Stat(t)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("'%s': does not exist", t)
			}
			return fmt.Errorf("failed to check '%s' existence: %w", t, err)
		}
		if fileInfo.IsDir() {
			return zipDir(zipFilePath, t)
		}
		return zipFiles(zipFilePath, "", []string{t})
	case []string:
		return zipFiles(zipFilePath, "", t)
	}
	return nil
}

func zipDir(zipFilePath, dir string) error {
	filesToZip := make([]string, 0)
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to open '%s': %w", path, err)
		}
		if !info.IsDir() {
			filesToZip = append(filesToZip, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return zipFiles(zipFilePath, filepath.Dir(dir), filesToZip)
}

func zipFiles(zipFilePath string, baseDir string, filesToZip []string) error {
	if err := os.MkdirAll(filepath.Dir(zipFilePath), FileMode_rwxrwxrwx); err != nil {
		return err
	}
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Create a new zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add files to the zip archive
	for _, file := range filesToZip {
		var relativeDir string
		if len(baseDir) > 0 {
			relPath, err := filepath.Rel(baseDir, file)
			if err != nil {
				return err
			}
			relativeDir = filepath.Dir(relPath)
		}
		if err := addFileToZip(zipWriter, relativeDir, file); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, relativeDir, filePath string) error {
	// Open the file to be added to the zip archive
	fileToZip, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get file info
	fileInfo, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	// Create a new file header
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return err
	}

	// Specify the name of the file in the zip archive
	header.Name = filepath.Join(relativeDir, filepath.Base(filePath))
	header.Method = zip.Deflate

	// Create a new zip file entry
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// Copy the file data to the zip file entry
	if _, err = io.Copy(writer, fileToZip); err != nil {
		return err
	}
	return nil
}

func Unzip(zipFileName, destDir string) error {
	reader, err := zip.OpenReader(zipFileName)
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()

	// Iterate through each file in the zip archive
	for _, file := range reader.File {
		// Open the file from the zip archive
		zippedFile, err := file.Open()
		if err != nil {
			return err
		}

		// Create the corresponding file on the disk
		extractedFilePath := filepath.Join(destDir, file.Name) // nolint
		if err := os.MkdirAll(filepath.Dir(extractedFilePath), FileMode_rwxrwxrwx); err != nil {
			return err
		}
		extractedFile, err := os.Create(extractedFilePath)
		if err != nil {
			return err
		}

		// nolint
		if _, err = io.Copy(extractedFile, zippedFile); err != nil {
			return err
		}

		if err := extractedFile.Close(); err != nil {
			return err
		}
		if err := zippedFile.Close(); err != nil {
			return err
		}
	}
	return nil
}
