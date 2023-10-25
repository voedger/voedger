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

// Creates and returns new comment.
func makeComment(v ...string) comment {
	return comment{c: strings.Join(v, "\n")}
}

func (c *comment) Comment() string {
	return c.c
}

func (c *comment) CommentLines() []string {
	return strings.Split(c.c, "\n")
}

func (c *comment) SetComment(v ...string) {
	c.c = strings.Join(v, "\n")
}

type nullComment struct{}

func (c *nullComment) Comment() string        { return "" }
func (c *nullComment) CommentLines() []string { return []string{} }
