/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import "github.com/voedger/voedger/pkg/parser"

func Provide() parser.PackageFS {
	return parser.PackageFS{
		Path: ClusterPackage,
		FS:   schemaFS,
	}
}
