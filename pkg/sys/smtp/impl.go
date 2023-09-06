/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package smtp

import "strings"

func (c Cfg) GetFrom() string {
	return strings.ReplaceAll(c.Username, "mailto:", "")
}
