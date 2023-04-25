/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package sqlschema

import fs "io/fs"

type IReadFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

// TODO: in: FS, moduleCache [string]FS, out: (Schema, DependencySchemas [string]Schema, error)

type FSParser func(fs IReadFS, subDir string) (*SchemaAST, error)

// TODO: func(fileName string, content string) (*SchemaAST, error)
type StringParser func(string) (*SchemaAST, error)
