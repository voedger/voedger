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
	err = MergePackageSchemas([]*PackageSchemaAST{pkgExample})
	require.NoError(t, err)

}

func Test_Expressions(t *testing.T) {
	require := require.New(t)

	_, err := ParseFile("file1.sql", `SCHEMA test; 
	TABLE MyTable(
		Int1 text DEFAULT 1 CHECK(Int1 > Int2),
		Int1 int DEFAULT 1 CHECK(Text != "asd"),
		Int1 int DEFAULT 1 CHECK(Int2 > -5),
		Int1 int DEFAULT 1 CHECK(TextField > "asd" AND (SomeFloat/3.2)*4 != 5.003),
		Int1 int DEFAULT 1 CHECK(SomeFunc("a", TextField) AND BoolField=FALSE),
		
		CHECK(MyRowValidator(this))
	)
	`)
	require.NoError(err)

}

func Test_Duplicates(t *testing.T) {
	require := require.New(t)

	ast1, err := ParseFile("file1.sql", `SCHEMA test; 
	EXTENSION ENGINE BUILTIN (
		FUNCTION MyTableValidator() RETURNS void;
		FUNCTION MyTableValidator(TableRow) RETURNS string;	
		FUNCTION MyFunc2() RETURNS void;
	)
	`)
	require.NoError(err)

	ast2, err := ParseFile("file2.sql", `SCHEMA test; 
	WORKSPACE ChildWorkspace (
		TAG MyFunc2; -- duplicate
		EXTENSION ENGINE BUILTIN (
			FUNCTION MyFunc3() RETURNS void;
			FUNCTION MyFunc4() RETURNS void;
		);
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
		"file1.sql:4:3: MyTableValidator redeclared",
		"file2.sql:3:3: MyFunc2 redeclared",
		"file2.sql:9:4: MyFunc4 redeclared",
	}, "\n"))

}

func Test_Comments(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	EXTENSION ENGINE BUILTIN (
		-- My function
		-- line 2
		FUNCTION MyFunc() RETURNS void;
	);
	`)
	require.Nil(err)

	ps, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	require.NotNil(ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments)
	require.Equal(2, len(ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments))
	require.Equal("My function", ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments[0])
	require.Equal("line 2", ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments[1])
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

func Test_Undefined(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	WORKSPACE test (
		EXTENSION ENGINE WASM (
			COMMAND Orders() WITH Tags=[UndefinedTag];
			QUERY Query1 RETURNS text WITH Rate=UndefinedRate, Comment=xyz.UndefinedComment;
			PROJECTOR ON COMMAND xyz.CreateUPProfile AFFECTS sys.HTTPStorage;
		)
	)
	`)
	require.Nil(err)

	pkg, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	err = MergePackageSchemas([]*PackageSchemaAST{pkg})

	require.EqualError(err, strings.Join([]string{
		"example.sql:4:4: UndefinedTag undefined",
		"example.sql:5:4: UndefinedRate undefined",
		"example.sql:5:4: xyz undefined",
		"example.sql:6:4: xyz undefined",
	}, "\n"))
}

func Test_Imports(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA pkg1;
	IMPORT SCHEMA "github.com/untillpro/airsbp3/pkg2";
	IMPORT SCHEMA "github.com/untillpro/airsbp3/pkg3" AS air;
	WORKSPACE test (
		EXTENSION ENGINE WASM (
    		COMMAND Orders WITH Tags=[pkg2.SomeTag];
    		QUERY Query1 RETURNS text WITH Comment=pkg2.SomeComment;
    		QUERY Query2 RETURNS text WITH Comment=air.SomeComment;
    		QUERY Query3 RETURNS text WITH Comment=air.SomeComment2; -- air.SomeComment2 undefined
    		PROJECTOR ON COMMAND Air.CreateUPProfil AFFECTS sys.HTTPStorage; -- Air undefined
		)
	)
	`)
	require.NoError(err)
	pkg1, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg1", []*FileSchemaAST{fs})
	require.NoError(err)

	fs, err = ParseFile("example.sql", `SCHEMA pkg2;
	TAG SomeTag;
	COMMENT SomeComment "Hello world!";
	`)
	require.NoError(err)
	pkg2, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg2", []*FileSchemaAST{fs})
	require.NoError(err)

	fs, err = ParseFile("example.sql", `SCHEMA pkg3;
	COMMENT SomeComment "Hello world!";
	`)
	require.NoError(err)
	pkg3, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg3", []*FileSchemaAST{fs})
	require.NoError(err)

	err = MergePackageSchemas([]*PackageSchemaAST{pkg1, pkg2, pkg3})
	require.EqualError(err, strings.Join([]string{
		"example.sql:9:7: air.SomeComment2 undefined",
		"example.sql:10:7: Air undefined",
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
