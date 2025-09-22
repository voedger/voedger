/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package filesu_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/voedger/voedger/pkg/goutils/filesu"
)

func ExampleCopyFile() {
	// Create a temporary source file
	srcDir := os.TempDir()
	srcFile := filepath.Join(srcDir, "source.txt")
	err := os.WriteFile(srcFile, []byte("Hello, World!"), 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(srcFile)

	// Create destination directory
	dstDir := filepath.Join(os.TempDir(), "destination")
	defer os.RemoveAll(dstDir)

	// Copy file to destination directory
	err = filesu.CopyFile(srcFile, dstDir)
	if err != nil {
		log.Fatal(err)
	}

	// Verify the file was copied
	dstFile := filepath.Join(dstDir, "source.txt")
	content, err := os.ReadFile(dstFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(content))
	// Output: Hello, World!
}

func ExampleCopyDir() {
	// Create a temporary source directory with files
	srcDir := filepath.Join(os.TempDir(), "source_dir")
	err := os.MkdirAll(srcDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	// Create some files in source directory
	err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("Content 1"), 0644)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("Content 2"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Create destination directory
	dstDir := filepath.Join(os.TempDir(), "destination_dir")
	defer os.RemoveAll(dstDir)

	// Copy entire directory
	err = filesu.CopyDir(srcDir, dstDir)
	if err != nil {
		log.Fatal(err)
	}

	// Verify files were copied
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil {
		log.Fatal(err)
	}
	content2, err := os.ReadFile(filepath.Join(dstDir, "file2.txt"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(content1))
	fmt.Println(string(content2))
	// Output: Content 1
	// Content 2
}

func ExampleExists() {
	// Check if a file exists
	tempFile := filepath.Join(os.TempDir(), "test_file.txt")

	// File doesn't exist yet
	exists, err := filesu.Exists(tempFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File exists before creation: %t\n", exists)

	// Create the file
	err = os.WriteFile(tempFile, []byte("test content"), 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tempFile)

	// Check again after creation
	exists, err = filesu.Exists(tempFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File exists after creation: %t\n", exists)

	// Output: File exists before creation: false
	// File exists after creation: true
}
