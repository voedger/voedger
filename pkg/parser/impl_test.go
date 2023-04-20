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

//go:embed example_app/expectedParsed.schema
var expectedParsedExampledSchemaStr string

func Test_BasicUsage(t *testing.T) {

	parser := ProvideEmbedParser()
	parsedSchema, err := parser(fs, "example_app")
	require.NoError(t, err)

	parsedSchemaStr := repr.String(parsedSchema, repr.Indent(" "))
	fmt.Println(parsedSchemaStr)

	require.Equal(t, expectedParsedExampledSchemaStr, parsedSchemaStr)
}

func Test_Duplicates(t *testing.T) {
	require := require.New(t)

	parser := ProvideStringParser()

	_, err := parser(`SCHEMA test; 
	FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
	FUNCTION MyTableValidator(TableRow) RETURNS string ENGINE WASM;		
	`)

	require.ErrorContains(err, "schema test contains duplicated name MyTableValidator")
}
