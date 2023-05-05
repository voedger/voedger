/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import "strings"

func IsBlank(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}
