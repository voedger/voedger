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
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/parser"
)

//go:embed testdata/sys.old.vsql
var oldSysSchemaFS embed.FS

//go:embed testdata/pkg1.old.vsql
var oldPkg1SchemaFS embed.FS

//go:embed testdata/pkg2.vsql
var pkg2SchemaFS embed.FS

//go:embed testdata/sys.new.vsql
var newSysSchemaFS embed.FS

//go:embed testdata/pkg1.new.vsql
var newPkg1SchemaFS embed.FS

func Test_Basic(t *testing.T) {
	oldSysPkgAST, err := parser.ParsePackageDir(appdef.SysPackage, oldSysSchemaFS, "testdata")
	require.NoError(t, err)

	oldPkg1AST, err := parser.ParsePackageDir("pkg1", oldPkg1SchemaFS, "testdata")
	require.NoError(t, err)

	oldpkg2AST, err := parser.ParsePackageDir("pkg2", pkg2SchemaFS, "testdata")
	require.NoError(t, err)

	newpkg2AST, err := parser.ParsePackageDir("pkg2", pkg2SchemaFS, "testdata")
	require.NoError(t, err)

	oldPackages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{oldSysPkgAST, oldPkg1AST, oldpkg2AST})
	require.NoError(t, err)

	newSysPkgAST, err := parser.ParsePackageDir(appdef.SysPackage, newSysSchemaFS, "testdata")
	require.NoError(t, err)

	newPkg1AST, err := parser.ParsePackageDir("pkg1", newPkg1SchemaFS, "testdata")
	require.NoError(t, err)

	newPackages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{newSysPkgAST, newPkg1AST, newpkg2AST})
	require.NoError(t, err)

	require.Equal(t, oldPackages.Name, newPackages.Name)

	oldBuilder := builder.New()
	require.NoError(t, parser.BuildAppDefs(oldPackages, oldBuilder))

	newBuilder := builder.New()
	require.NoError(t, parser.BuildAppDefs(newPackages, newBuilder))

	oldAppDef, err := oldBuilder.Build()
	require.NoError(t, err)

	newAppDef, err := newBuilder.Build()
	require.NoError(t, err)

	t.Run("CheckBackwardCompatibility", func(t *testing.T) {
		expectedErrors := []CompatibilityError{
			{OldTreePath: []string{"AppDef", "Types", "sys.Profile", "Types", "sys.ProfileTable"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.ProfileTable"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeTable"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Password"}, ErrorType: ErrorTypeOrderChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Email"}, ErrorType: ErrorTypeOrderChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginParams", "Fields", "Login"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginParams", "Fields", "ProfileCluster"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.CreateLoginParams", "Fields", "ProfileToken"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.AnotherOneTable", "Fields"}, ErrorType: ErrorTypeNodeInserted},
			{OldTreePath: []string{"AppDef", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeOrderChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeWorkspace", "Types", "sys.SomeTable"}, ErrorType: ErrorTypeNodeRemoved},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeCommand", "CommandArgs"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeCommand", "UnloggedArgs"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeCommand", "CommandResult"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.AbsWorkspace", "Abstract"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeView", "PartKeyFields"}, ErrorType: ErrorTypeNodeModified},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeView", "Fields", "E"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.SomeView", "ClustColsFields", "B"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Types", "sys.AnotherOneTable", "Uniques", "sys.AnotherOneTable$uniques$01", "UniqueFields"}, ErrorType: ErrorTypeNodeModified},
			{OldTreePath: []string{"AppDef", "Packages", "pkg1"}, ErrorType: ErrorTypeValueChanged},
			{OldTreePath: []string{"AppDef", "Packages", "pkg2"}, ErrorType: ErrorTypeValueChanged},
		}
		compatErrors := CheckBackwardCompatibility(oldAppDef, newAppDef)
		fmt.Println(compatErrors.Error())
		validateCompatibilityErrors2(t, expectedErrors, compatErrors)
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

func validateCompatibilityErrors2(t *testing.T, expectedErrors []CompatibilityError, actualErrors *CompatibilityErrors) {
	for _, actualErr := range actualErrors.Errors {
		require.True(t, slices.ContainsFunc(expectedErrors, func(expectedErr CompatibilityError) bool {
			if expectedErr.Path() == actualErr.Path() && expectedErr.ErrorType == actualErr.ErrorType {
				return true
			}
			return false
		}), actualErr.Error())
	}
	for _, expectedErr := range expectedErrors {
		require.True(t, slices.ContainsFunc(actualErrors.Errors, func(actualErr CompatibilityError) bool {
			if expectedErr.Path() == actualErr.Path() && expectedErr.ErrorType == actualErr.ErrorType {
				return true
			}
			return false
		}), "error is expected but not occurred: "+expectedErr.Error())
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
