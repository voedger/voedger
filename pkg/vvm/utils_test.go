/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package vvm

import (
	"embed"
)

//go:embed testcontent/subdir/*
var testContentFS embed.FS

// func TestSchemaFilesContent(t *testing.T) {
// 	require := require.New(t)
// 	ep := extensionpoints.NewRootExtensionPoint()
// 	packageFullyQualifiedName := "github.com/voedger/voedger/pkg/vvm/packagename"
// 	apps.RegisterSchemaFS(testContentFS, packageFullyQualifiedName, ep)
// 	content, err := SchemaFilesContent(ep, "testcontent/subdir")
// 	require.NoError(err)

// 	expectedContent := []string{"hello world", "test test"}
// 	actualContent := []string{}
// 	for k, v := range content[packageFullyQualifiedName] {
// 		fmt.Println(k)
// 		actualContent = append(actualContent, string(v))
// 	}
// 	require.Equal(len(expectedContent), len(actualContent))
// 	for _, v := range expectedContent {
// 		require.Contains(actualContent, v)
// 	}
// }
