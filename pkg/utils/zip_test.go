/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZipAndUnzip(t *testing.T) {
	require := require.New(t)

	content := `Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
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
	tmpDir := t.TempDir()
	filePaths := [][]string{{"file1.txt"}, {"file2.txt"}, {"subdir", "file3.txt"}}

	filesToZip := make([]string, 0, len(filePaths))
	for _, filePath := range filePaths {
		filesToZip = append(filesToZip, filepath.Join(tmpDir, filepath.Join(filePath...)))

		err := os.MkdirAll(filepath.Join(tmpDir, filepath.Join(filePath[:len(filePath)-1]...)), FileMode_rwxrwxrwx)
		require.NoError(err)
		err = os.WriteFile(filepath.Join(tmpDir, filepath.Join(filePath...)), []byte(content), FileMode_rw_rw_rw_)
		require.NoError(err)
	}

	// Test zipping and unzipping
	zipFilePath := filepath.Join(tmpDir, "test.zip")
	destDir := filepath.Join(tmpDir, "unzipped")
	err := os.Mkdir(destDir, FileMode_rwxrwxrwx)
	require.NoError(err)

	// Test zipping
	err = Zip(zipFilePath, filesToZip)
	require.NoError(err)

	// Test unzipping
	err = Unzip(zipFilePath, destDir)
	require.NoError(err)

	// Check if unzipped files match the original files
	for _, filePath := range filePaths {
		filename := filepath.Base(filepath.Join(filePath...))
		unzippedContent, err := os.ReadFile(filepath.Join(tmpDir, filepath.Join(filePath...)))
		require.NoError(err)
		require.Equal(string(unzippedContent), content, "Unzipped file content for %s doesn't match", filename)
	}
}
