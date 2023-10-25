/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"embed"
	"path"
	"path/filepath"
	"runtime"

	"github.com/voedger/voedger/pkg/extensionpoints"
)

func RegisterSchemaFS(fsi embed.FS, packageName string, ep extensionpoints.IExtensionPoint) {
	_, file, _, _ := runtime.Caller(1)
	dir := filepath.Dir(file)
	epSqlFiles := ep.ExtensionPoint(EPSchemasFS)
	epPackage := epSqlFiles.ExtensionPoint(packageName)
	dirWithPackageName := path.Join(dir, packageName) // if package distributed among several directories
	epPackage.AddNamed(dirWithPackageName, fsi)
}
