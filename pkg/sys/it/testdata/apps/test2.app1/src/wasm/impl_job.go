/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package main

import (
	ext "github.com/voedger/voedger/pkg/exttinygo"
)

//export Job1_sidecar
func Job1_sidecar() {
	log := ext.KeyBuilder(ext.StorageLogger, ext.NullEntity)
	log.PutInt32("LogLevel", 3) // LogLevelInfo
	ext.NewValue(log).PutString("Message", "Job done")
}
