/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestZipAndUnzip(t *testing.T) {
	require := require.New(t)
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	unzippedDir := filepath.Join(tmpDir, "unzipped")
	zipFilePath := filepath.Join(srcDir, "test.zip") // place the zip file into the dir that is zipping to check if the target zip file itself is not included to the zip
	filesToZip := []string{
		"file1.txt",
		"file2.txt",
		filepath.Join("subdir", "file3.txt"),
	}

	for _, fileToZip := range filesToZip {
		err := os.MkdirAll(filepath.Join(srcDir, filepath.Dir(fileToZip)), FileMode_rwxrwxrwx)
		require.NoError(err)
		err = os.WriteFile(filepath.Join(srcDir, fileToZip), []byte(content), FileMode_rw_rw_rw_)
		require.NoError(err)
	}

	// Test zipping and unzipping
	err := os.Mkdir(unzippedDir, FileMode_rwxrwxrwx)
	require.NoError(err)

	// Test zipping
	err = Zip(srcDir, zipFilePath)
	require.NoError(err)

	// Test unzipping
	err = Unzip(zipFilePath, unzippedDir)
	require.NoError(err)

	// check file content
	for _, expectedFile := range filesToZip {
		actualContent, err := os.ReadFile(filepath.Join(unzippedDir, expectedFile))
		require.NoError(err)
		require.Equal(content, string(actualContent))
	}

	// check there are no unexpected unzipped files
	err = filepath.WalkDir(unzippedDir, func(unzippedFile string, d fs.DirEntry, err error) error {
		require.NoError(err)
		if d.IsDir() {
			return nil
		}
		actualRelFile, err := filepath.Rel(unzippedDir, unzippedFile)
		require.NoError(err)
		require.Contains(filesToZip, actualRelFile)
		return nil
	})
	require.NoError(err)

	t.Run("target zip already exists", func(t *testing.T) {
		err = Zip(srcDir, zipFilePath)
		require.ErrorContains(err, "exists already")
	})

}

const content = `Lorem ipsum dolor sit amet, consectetur adipiscing elit.
 Sed ut urna at nibh efficitur tristique nec eget odio. Fusce ac volutpat arcu, eu venenatis nulla.
 Quisque condimentum libero id bibendum tincidunt. Maecenas consectetur sapien ut enim vehicula, non fermentum odio vehicula.
 Nullam id felis eleifend, ullamcorper justo non, placerat sapien. Integer ac convallis velit.
 Integer semper dui eget erat sodales bibendum. Vivamus interdum gravida libero, eget tempus purus ultrices nec.
 Ut vel tellus a nisl vehicula cursus a non urna. Mauris sed felis elit. Sed et consectetur odio.
 Proin lacinia ligula at elit aliquet, a lobortis elit dictum. Ut euismod tincidunt nisi.
 Aenean malesuada nisi non nisl dictum vestibulum. Lorem ipsum dolor sit amet, consectetur adipiscing elit.
 Sed ut urna at nibh efficitur tristique nec eget odio. Fusce ac volutpat arcu, eu venenatis nulla.
 Quisque condimentum libero id bibendum tincidunt. Maecenas consectetur sapien ut enim vehicula, non fermentum odio vehicula.
 Nullam id felis eleifend, ullamcorper justo non, placerat sapien. Integer ac convallis velit.
 Integer semper dui eget erat sodales bibendum. Vivamus interdum gravida libero, eget tempus purus ultrices nec.
 Ut vel tellus a nisl vehicula cursus a non urna. Mauris sed felis elit. Sed et consectetur odio.
 Proin lacinia ligula at elit aliquet, a lobortis elit dictum. Ut euismod tincidunt nisi.
 Aenean malesuada nisi non nisl dictum vestibulum. Lorem ipsum dolor sit amet, consectetur adipiscing elit.
 Sed ut urna at nibh efficitur tristique nec eget odio. Fusce ac volutpat arcu, eu venenatis nulla.
 Quisque condimentum libero id bibendum tincidunt. Maecenas consectetur sapien ut enim vehicula, non fermentum odio vehicula.
 Nullam id felis eleifend, ullamcorper justo non, placerat sapien. Integer ac convallis velit.
 Integer semper dui eget erat sodales bibendum. Vivamus interdum gravida libero, eget tempus purus ultrices nec.
 Ut vel tellus a nisl vehicula cursus a non urna. Mauris sed felis elit. Sed et consectetur odio.
 Proin lacinia ligula at elit aliquet, a lobortis elit dictum. Ut euismod tincidunt nisi.
 Aenean malesuada nisi non nisl dictum vestibulum`
