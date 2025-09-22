/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package internal

import (
	"os"
	"strings"
)

func IsDebug() bool {
	return strings.Contains(os.Args[0], "__debug_bin")
}
