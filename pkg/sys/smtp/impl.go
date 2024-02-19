/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package smtp

import "strings"

func (c Cfg) GetFrom() string {
	if len(c.From) == 0 {
		return strings.ReplaceAll(c.Username, "mailto:", "")
	}
	return c.From
}
