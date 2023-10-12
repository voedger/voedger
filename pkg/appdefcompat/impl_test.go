/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/require"

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

	expectedErrors := []CompatibilityError{
		{OldTreePath: []string{"AppDef", "Types", "sys.Profile", "Types", "sys.ProfileTable"}, ErrorType: ErrorTypeNodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Password"}, ErrorType: ErrorTypeOrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Email"}, ErrorType: ErrorTypeOrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "Login"}, ErrorType: ErrorTypeNodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileCluster"}, ErrorType: ErrorTypeValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileToken"}, ErrorType: ErrorTypeValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields"}, ErrorType: ErrorTypeNodeInserted},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeOrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.SomeTable"}, ErrorType: ErrorTypeNodeRemoved},
	}
	compatErrors := CheckBackwardCompatibility(oldAppDef, newAppDef)
	validateCompatibilityErrors(t, expectedErrors, compatErrors)

	// testing ignoring some compatibility errors
	expectedFilteredErrors := []CompatibilityError{
		{OldTreePath: []string{"AppDef", "Types", "sys.Profile", "Types", "sys.ProfileTable"}, ErrorType: ErrorTypeNodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Password"}, ErrorType: ErrorTypeOrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Email"}, ErrorType: ErrorTypeOrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "Login"}, ErrorType: ErrorTypeNodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileCluster"}, ErrorType: ErrorTypeValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields"}, ErrorType: ErrorTypeNodeInserted},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrorType: ErrorTypeOrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.SomeTable"}, ErrorType: ErrorTypeNodeRemoved},
	}

	pathsToIgnore := [][]string{
		{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileToken"},
	}
	filteredCompatErrors := IgnoreCompatibilityErrors(compatErrors, pathsToIgnore)
	validateCompatibilityErrors(t, expectedFilteredErrors, filteredCompatErrors)
}

func validateCompatibilityErrors(t *testing.T, expectedErrors []CompatibilityError, compatErrors *CompatibilityErrors) {
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
}
