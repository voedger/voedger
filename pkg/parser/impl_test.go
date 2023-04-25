/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"embed"
	"fmt"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"
)

//go:embed example_app/*.sql
var efs embed.FS

//_go:embed example_app/expectedParsed.schema
//var expectedParsedExampledSchemaStr string

func Test_BasicUsage(t *testing.T) {

	parsedSchema, err := ParsePackageDir("github.com/untillpro/exampleschema", efs, "example_app")
	require.NoError(t, err)

	parsedSchemaStr := repr.String(parsedSchema, repr.Indent(" "), repr.IgnorePrivate())
	fmt.Println(parsedSchemaStr)

	//require.Equal(t, expectedParsedExampledSchemaStr, parsedSchemaStr)
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

	_, err = mergeFileSchemaASTsImpl("", []*FileSchemaAST{ast1, ast2})

	// TODO: use golang messages like
	// ./types2.go:17:7: EmbedParser redeclared
	//     ./types.go:17:6: other declaration of EmbedParser
	require.ErrorContains(err, "file1.sql:3:2: MyTableValidator redeclared")
	require.ErrorContains(err, "file2.sql:3:3: MyFunc2 redeclared")
	require.ErrorContains(err, "file2.sql:7:4: MyFunc4 redeclared")
}

func Test_Comments(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	-- My function
	-- line 2
	FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
	`)

	require.Nil(err)

	ps, err := mergeFileSchemaASTsImpl("", []*FileSchemaAST{fs})

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

	_, err = mergeFileSchemaASTsImpl("", []*FileSchemaAST{ast1, ast2})
	require.ErrorContains(err, "file2.sql: package schema2; expected schema1")
}
