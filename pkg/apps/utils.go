/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"embed"

	"github.com/voedger/voedger/pkg/extensionpoints"
)

func RegisterSchemaFS(fs embed.FS, packageFQN string, ep extensionpoints.IExtensionPoint) {
	epSqlFiles := ep.ExtensionPoint(EPSchemasFS)
	epPackage := epSqlFiles.ExtensionPoint(packageFQN)
	epPackage.Add(fs)
}
