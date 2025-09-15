/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package filesu

import "io/fs"

type IReadFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

type copyOpts struct {
	fm             fs.FileMode
	skipExisting   bool
	files          []string
	targetFileName string
}

type CopyOpt func(co *copyOpts)
