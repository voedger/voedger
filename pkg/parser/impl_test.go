/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import (
	"embed"
	"fmt"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"
)

//go:embed example_app/*.sql
var fs embed.FS

//_go:embed example_app/expectedParsed.schema
//var expectedParsedExampledSchemaStr string

func Test_BasicUsage(t *testing.T) {

	parser := ProvideEmbedParser()
	parsedSchema, err := parser(fs, "example_app")
	require.NoError(t, err)

	parsedSchemaStr := repr.String(parsedSchema, repr.Indent(" "), repr.IgnorePrivate())
	fmt.Println(parsedSchemaStr)

	//require.Equal(t, expectedParsedExampledSchemaStr, parsedSchemaStr)
}

func Test_Duplicates(t *testing.T) {
	require := require.New(t)

	parser := ProvideStringParser()

	_, err := parser(`SCHEMA test; 
	FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
	FUNCTION MyTableValidator(TableRow) RETURNS string ENGINE WASM;	
	FUNCTION MyFunc2() RETURNS void ENGINE BUILTIN;
	WORKSPACE ChildWorkspace (
		TAG MyFunc2; -- duplicate
		FUNCTION MyFunc3() RETURNS void ENGINE BUILTIN;
		FUNCTION MyFunc4() RETURNS void ENGINE BUILTIN;
		WORKSPACE InnerWorkspace (
			ROLE MyFunc4; -- duplicate
		)
	)
	`)

	require.ErrorContains(err, "3:2: schema 'test' contains duplicated name MyTableValidator")
	require.ErrorContains(err, "6:3: schema 'test' contains duplicated name MyFunc2")
	require.ErrorContains(err, "10:4: schema 'test' contains duplicated name MyFunc4")
}

func Test_Comments(t *testing.T) {
	require := require.New(t)

	parser := ProvideStringParser()

	s, err := parser(`SCHEMA test; 
	-- My function
	-- line 2
	FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
	`)

	require.Nil(err)
	require.NotNil(s.Statements[0].Function.Comments)
	require.Equal(2, len(s.Statements[0].Function.Comments))
	require.Equal("My function", s.Statements[0].Function.Comments[0])
	require.Equal("line 2", s.Statements[0].Function.Comments[1])
}
