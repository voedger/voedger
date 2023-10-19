/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"embed"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/parser"
)

//go:embed sql/old.sql
var oldFS embed.FS

//go:embed sql/new.sql
var newFS embed.FS

func getSysPackageAST(file parser.IReadFS) *parser.PackageSchemaAST {
	pkgSys, err := parser.ParsePackageDir(appdef.SysPackage, file, "sql")
	if err != nil {
		panic(err)
	}
	return pkgSys
}

func Test_Basic(t *testing.T) {
	oldPackages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{
		getSysPackageAST(oldFS),
	})
	require.NoError(t, err)

	newPackages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{
		getSysPackageAST(newFS),
	})
	require.NoError(t, err)

	oldBuilder := appdef.New()
	require.NoError(t, parser.BuildAppDefs(oldPackages, oldBuilder))

	newBuilder := appdef.New()
	require.NoError(t, parser.BuildAppDefs(newPackages, newBuilder))

	oldAppDef, err := oldBuilder.Build()
	require.NoError(t, err)

	newAppDef, err := newBuilder.Build()
	require.NoError(t, err)

	t.Run("CheckBackwardCompatibility", func(t *testing.T) {
		expectedErrors := []CompatibilityError{
			{OldTreePath: []string{"AppDef", "Types", "sys.Profile", "Types", "sys.ProfileTable"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Password"}, ErrorType: ErrorTypeOrderChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Email"}, ErrorType: ErrorTypeOrderChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginParams", "Fields", "Login"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginParams", "Fields", "ProfileCluster"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginParams", "Fields", "ProfileToken"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.AnotherOneTable", "Fields"}, ErrorType: ErrorTypeNodeInserted},
			{OldTreePath: []string{"AppDef", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeOrderChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeWorkspace", "Types", "sys.SomeTable"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeCommand", "CommandArgs"}, ErrorType: ErrorTypeNodeModified},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeCommand", "CommandResult"}, ErrorType: ErrorTypeNodeModified},
			{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Abstract"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeView", "PartKeyFields"}, ErrorType: ErrorTypeNodeModified},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeView", "Fields", "E"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeView", "ClustColsFields", "B"}, ErrorType: ErrorTypeValueChanged},
		}
		allowedErrors := []CompatibilityError{
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeCommand", "UnloggedArgs"}},
		}
		allowedTypes := []string{
			"sys.NewTable",
			"sys.NewType",
			"sys.NewView",
			"sys.NewCommand",
			"sys.NewQuery",
			"sys.SomeQuery",
		}
		compatErrors := CheckBackwardCompatibility(oldAppDef, newAppDef)
		fmt.Println(compatErrors.Error())
		validateCompatibilityErrors(t, expectedErrors, allowedErrors, allowedTypes, compatErrors)
	})

	t.Run("IgnoreCompatibilityErrors", func(t *testing.T) {
		// testing ignoring some compatibility errors
		pathsToIgnore := [][]string{
			{"AppDef", "Types", "sys.CreateLoginParams", "Fields", "ProfileToken"},
		}
		compatErrors := CheckBackwardCompatibility(oldAppDef, newAppDef)
		fmt.Println(compatErrors.Error())
		filteredCompatErrors := IgnoreCompatibilityErrors(compatErrors, pathsToIgnore)
		checkPathsToIgnore(t, pathsToIgnore, compatErrors, filteredCompatErrors)
	})
}

func validateCompatibilityErrors(t *testing.T, expectedErrors []CompatibilityError, allowedErrors []CompatibilityError, allowedTypes []string, compatErrors *CompatibilityErrors) {
	for _, expectedErr := range expectedErrors {
		found := false
		for _, cerr := range compatErrors.Errors {
			if cerr.Path() == expectedErr.Path() && cerr.ErrorType == expectedErr.ErrorType {
				found = true
				break
			}
		}
		require.True(t, found, expectedErr.Error())
	}
	// allowed types
	for _, allowedType := range allowedTypes {
		found := false
		for _, compatErr := range compatErrors.Errors {
			if slices.Contains(compatErr.OldTreePath, allowedType) {
				found = true
				break
			}
		}
		require.False(t, found, fmt.Sprintf("type %s should be allowed", allowedType))
	}
	// allowed errors
	for _, allowedError := range allowedErrors {
		found := false
		allowedPath := allowedError.Path()
		for _, compatErr := range compatErrors.Errors {
			if strings.Contains(compatErr.Path(), allowedPath) {
				found = true
				break
			}
		}
		require.False(t, found, fmt.Sprintf("path %s should be allowed", allowedPath))
	}
}

func checkPathsToIgnore(t *testing.T, pathsToIgnore [][]string, compatErrors, filteredCompatErrors *CompatibilityErrors) {
	for _, pathToIgnore := range pathsToIgnore {
		found := false
		for _, cerr := range compatErrors.Errors {
			if cerr.Path() == strings.Join(pathToIgnore, pathDelimiter) {
				found = true
				break
			}
		}
		require.True(t, found, fmt.Sprintf("there is no path %s in compat errors", pathToIgnore))
	}
	for _, pathToIgnore := range pathsToIgnore {
		found := false
		for _, cerr := range filteredCompatErrors.Errors {
			if cerr.Path() == strings.Join(pathToIgnore, pathDelimiter) {
				found = true
				break
			}
		}
		require.False(t, found, fmt.Sprintf("path %s should be ignored", pathToIgnore))
	}
}
