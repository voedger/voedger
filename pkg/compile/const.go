/*
* Copyright (c) 2024-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package compile

const (
	DummyAppName      = "dummyApp"
	sysSchemaFileName = "sys.vsql"
	VoedgerPath       = "github.com/voedger/voedger"
	PkgDirName        = "pkg"
)

const (
	tmpSysGoModule = `package %s

import (
	_ "github.com/voedger/voedger/pkg/sys"
)

func init() {
}
`
)
