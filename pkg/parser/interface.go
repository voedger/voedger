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

// input:
//   modulesCache: key: module
// output:
//   schema - parsed package AST
//   deps - dependencies. Key is a fully-qualified package name
// type FSParser func(fs IReadFS, subDir string, modulesCache map[string]IReadFS) (schema *SchemaAST, deps map[string]*SchemaAST, err error)

type StringParser func(fileName string, content string) (*SchemaAST, error)
