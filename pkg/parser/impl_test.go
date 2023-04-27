/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed example_app/*.sql
var efs embed.FS

//_go:embed example_app/expectedParsed.schema
//var expectedParsedExampledSchemaStr string

func Test_BasicUsage(t *testing.T) {

	pkgExample, err := ParsePackageDir("github.com/untillpro/exampleschema", efs, "example_app")
	require.NoError(t, err)

	// := repr.String(pkgExample, repr.Indent(" "), repr.IgnorePrivate())
	//fmt.Println(parsedSchemaStr)

	// TODO: MergePackageSchemas should return ?.ISchema
	require.Nil(t, MergePackageSchemas([]*PackageSchemaAST{pkgExample}))

}

func Test_Duplicates(t *testing.T) {
	require := require.New(t)

	ast1, err := ParseFile("file1.sql", `SCHEMA test; 
	FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
	FUNCTION MyTableValidator(TableRow) RETURNS string ENGINE WASM;	
	FUNCTION MyFunc2() RETURNS void ENGINE BUILTIN;
	`)
	require.NoError(err)

	ast2, err := ParseFile("file2.sql", `SCHEMA test; 
	WORKSPACE ChildWorkspace (
		TAG MyFunc2; -- duplicate
		FUNCTION MyFunc3() RETURNS void ENGINE BUILTIN;
		FUNCTION MyFunc4() RETURNS void ENGINE BUILTIN;
		WORKSPACE InnerWorkspace (
			ROLE MyFunc4; -- duplicate
		)
	)
	`)
	require.NoError(err)

	_, err = MergeFileSchemaASTs("", []*FileSchemaAST{ast1, ast2})

	// TODO: use golang messages like
	// ./types2.go:17:7: EmbedParser redeclared
	//     ./types.go:17:6: other declaration of EmbedParser
	require.EqualError(err, strings.Join([]string{
		"file1.sql:3:2: MyTableValidator redeclared",
		"file2.sql:3:3: MyFunc2 redeclared",
		"file2.sql:7:4: MyFunc4 redeclared",
	}, "\n"))

}

func Test_Comments(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	-- My function
	-- line 2
	FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
	`)
	require.Nil(err)

	ps, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	require.NotNil(ps.Ast.Statements[0].Function.Comments)
	require.Equal(2, len(ps.Ast.Statements[0].Function.Comments))
	require.Equal("My function", ps.Ast.Statements[0].Function.Comments[0])
	require.Equal("line 2", ps.Ast.Statements[0].Function.Comments[1])
}

func Test_UnexpectedSchema(t *testing.T) {
	require := require.New(t)

	ast1, err := ParseFile("file1.sql", `SCHEMA schema1; ROLE abc;`)
	require.NoError(err)

	ast2, err := ParseFile("file2.sql", `SCHEMA schema2; ROLE xyz;`)
	require.NoError(err)

	_, err = MergeFileSchemaASTs("", []*FileSchemaAST{ast1, ast2})
	require.EqualError(err, "file2.sql: package schema2; expected schema1")
}

func Test_FunctionUndefined(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	WORKSPACE test (
    	COMMAND Orders AS SomeCmdFunc;
    	QUERY Query1 RETURNS text AS QueryFunc;
    	PROJECTOR ON COMMAND Air.CreateUPProfile AS Air.SomeProjectorFunc;
	)
	`)
	require.Nil(err)

	pkg, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	err = MergePackageSchemas([]*PackageSchemaAST{pkg})

	require.EqualError(err, strings.Join([]string{
		"example.sql:3:6: SomeCmdFunc undefined",
		"example.sql:4:6: QueryFunc undefined",
		"example.sql:5:6: Air undefined",
	}, "\n"))
}

func Test_MergePackageSchemas1(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA pkg1;
	IMPORT SCHEMA "github.com/untillpro/airsbp3/pkg2";
	IMPORT SCHEMA "github.com/untillpro/airsbp3/pkg3" AS air;
	WORKSPACE test (
    	COMMAND Orders AS pkg2.SomeCmdFunc;
    	QUERY Query1 RETURNS text AS pkg2.QueryFunc;
    	QUERY Query2 RETURNS text AS air.QueryFunc2;
    	PROJECTOR ON COMMAND Air.CreateUPProfile AS pkg2.SomeProjectorFunc2;
	)
	`)
	require.Nil(err)
	pkg1, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg1", []*FileSchemaAST{fs})
	require.Nil(err)

	fs, err = ParseFile("example.sql", `SCHEMA pkg2;
	FUNCTION SomeCmdFunc() RETURNS void ENGINE BUILTIN;
	FUNCTION QueryFunc() RETURNS void ENGINE BUILTIN;
	FUNCTION SomeProjectorFunc() RETURNS void ENGINE BUILTIN;
	`)
	require.Nil(err)
	pkg2, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg2", []*FileSchemaAST{fs})
	require.Nil(err)

	fs, err = ParseFile("example.sql", `SCHEMA pkg3;
	FUNCTION QueryFunc2() RETURNS text ENGINE BUILTIN;
	`)
	require.Nil(err)
	pkg3, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg3", []*FileSchemaAST{fs})
	require.Nil(err)

	err = MergePackageSchemas([]*PackageSchemaAST{pkg1, pkg2, pkg3})
	require.EqualError(err, strings.Join([]string{
		"example.sql:6:6: function result do not match",
		"example.sql:8:6: pkg2.SomeProjectorFunc2 undefined",
	}, "\n"))

}

func Test_AbstractWorkspace(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	WORKSPACE ws1 ();
	ABSTRACT WORKSPACE ws2();
	ABSTRACT WORKSPACE ws3();
	WORKSPACE ws4 OF ws2,test.ws3 ();
	`)
	require.Nil(err)

	ps, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	require.False(ps.Ast.Statements[0].Workspace.Abstract)
	require.True(ps.Ast.Statements[1].Workspace.Abstract)
	require.True(ps.Ast.Statements[2].Workspace.Abstract)
	require.False(ps.Ast.Statements[3].Workspace.Abstract)
	require.Equal("ws2", ps.Ast.Statements[3].Workspace.Of[0].String())
	require.Equal("test.ws3", ps.Ast.Statements[3].Workspace.Of[1].String())

}
