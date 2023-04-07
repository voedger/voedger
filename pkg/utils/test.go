/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package coreutils

import (
	"os"
	"strings"
)

func IsTest() bool {
	return strings.Contains(os.Args[0], ".test") || IsDebug()
}

func IsDebug() bool {
	return strings.Contains(os.Args[0], "__debug_bin")
}
