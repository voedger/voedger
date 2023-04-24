/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package sqlschema

import "embed"

// TODO: why embed.FS, how to process a normal folder?
// TODO: FSParser()
type EmbedParser func(fs embed.FS, subDir string) (*SchemaAST, error)

// TODO: func(fileName string, content string) (*SchemaAST, error)
type StringParser func(string) (*SchemaAST, error)
