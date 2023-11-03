/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"embed"

	"github.com/voedger/voedger/pkg/extensionpoints"
)

func RegisterSchemaFS(fsi embed.FS, packageFQN string, ep extensionpoints.IExtensionPoint) {
	// _, file, _, _ := runtime.Caller(1)
	// dir := filepath.Dir(file)
	epSqlFiles := ep.ExtensionPoint(EPSchemasFS)
	epPackage := epSqlFiles.ExtensionPoint(packageFQN)
	epPackage.Add(fsi)
	// dirWithPackageName := path.Join(dir, packageName) // if package distributed among several directories
	// epPackage.AddNamed(packageFQN, fsi)
}
