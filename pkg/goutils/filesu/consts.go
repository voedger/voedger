/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package filesu

import "io/fs"

const (
	FileMode_DefaultForDir  fs.FileMode = 0777 // rwxrwxrwx
	FileMode_DefaultForFile fs.FileMode = 0666 // rw_rw_rw_
)
