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

	expectedErrors := []CompatibilityError{
		{OldTreePath: []string{"AppDef", "Types", "sys.Profile", "Types", "sys.ProfileTable"}, ErrMessage: NodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Password"}, ErrMessage: OrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Email"}, ErrMessage: OrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "Login"}, ErrMessage: NodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileCluster"}, ErrMessage: ValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileToken"}, ErrMessage: ValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "D"}, ErrMessage: NodeInserted},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrMessage: ValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrMessage: OrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.SomeTable"}, ErrMessage: NodeRemoved},
	}
	compatErrors, err := CheckBackwardCompatibility(oldBuilder, newBuilder)
	require.NoError(t, err)
	validateCompatibilityErrors(t, expectedErrors, compatErrors)

	// testing ignoring some compatibility errors
	expectedErrors = []CompatibilityError{
		{OldTreePath: []string{"AppDef", "Types", "sys.Profile", "Types", "sys.ProfileTable"}, ErrMessage: NodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Password"}, ErrMessage: OrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginUnloggedParams", "Fields", "Email"}, ErrMessage: OrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "Login"}, ErrMessage: NodeRemoved},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileCluster"}, ErrMessage: ValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "D"}, ErrMessage: NodeInserted},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrMessage: ValueChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.AnotherOneTable", "Fields", "C"}, ErrMessage: OrderChanged},
		{OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.SomeTable"}, ErrMessage: NodeRemoved},
	}

	toBeIgnored := []CompatibilityError{
		{
			OldTreePath: []string{"AppDef", "Types", "sys.Workspace", "Types", "sys.CreateLoginParams", "Fields", "ProfileToken"},
			ErrMessage:  ValueChanged,
		},
	}
	filteredCompatErrors := IgnoreCompatibilityErrors(compatErrors, toBeIgnored)
	validateCompatibilityErrors(t, expectedErrors, filteredCompatErrors)
}

func validateCompatibilityErrors(t *testing.T, expectedErrors []CompatibilityError, compatErrors *CompatibilityErrors) {
	found := false
	for _, expectedErr := range expectedErrors {
		for _, cerr := range compatErrors.Errors {
			if cerr.Path() == expectedErr.Path() && cerr.ErrMessage == expectedErr.ErrMessage {
				found = true
				break
			}
		}
		require.True(t, found, expectedErr.Error())
	}
}
