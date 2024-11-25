/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package main

import (
	"strconv"

	ext "github.com/voedger/voedger/pkg/exttinygo"
)

//export Job1_sidecar
func Job1_sidecar() {
	var counter int32
	viewKey := ext.KeyBuilder(ext.StorageView, "github.com/voedger/sidecartestapp.JobStateView")
	viewKey.PutInt32("Pk", 1)
	viewKey.PutInt32("Cc", 1)
	viewValue, exists := ext.QueryValue(viewKey)
	if exists {
		counter = viewValue.AsInt32("Counter")
	}
	counter++

	logKey := ext.KeyBuilder(ext.StorageLogger, ext.NullEntity)
	logKey.PutInt32("LogLevel", int32(3)) // LogLevelInfo

	logMsg := ext.NewValue(logKey)
	logMsg.PutString("Message", "job:"+strconv.Itoa(int(counter)))

	newValue := ext.NewValue(viewKey)
	newValue.PutInt32("Counter", counter)

}
