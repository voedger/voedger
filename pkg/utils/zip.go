/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func Zip(zipFilePath string, filesToZip []string) error {
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = zipFile.Close()
	}()

	// Create a new zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		_ = zipWriter.Close()
	}()

	// Add files to the zip archive
	for _, file := range filesToZip {
		if err := addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, fileName string) error {
	// Open the file to be added to the zip archive
	fileToZip, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer func() {
		_ = fileToZip.Close()
	}()

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
	header.Name = fileName
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
		defer func() {
		}()

		// Create the corresponding file on the disk
		extractedFilePath := filepath.Join(destDir, filepath.Base(file.Name))
		extractedFile, err := os.Create(extractedFilePath)
		if err != nil {
			return err
		}

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
