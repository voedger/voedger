/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "strings"

// # Implements:
//  - IComment
//	- ICommentBuilder
type comment struct {
	c string
}

func (c *comment) Comment() string {
	return c.c
}

func (c *comment) SetComment(v ...string) {
	c.c = strings.Join(v, "\n")
}
