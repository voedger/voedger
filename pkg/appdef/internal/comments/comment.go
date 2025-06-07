/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package comments

import (
	"strings"
)

// # Supports:
//   - appdef.IWithComments
type WithComments struct {
	string
}

// Creates and returns new comment.
func MakeWithComments(v ...string) WithComments {
	return WithComments{strings.Join(v, "\n")}
}

func (c *WithComments) Comment() string {
	return c.string
}

func (c *WithComments) CommentLines() []string {
	if len(c.string) == 0 {
		return nil
	}
	return strings.Split(c.string, "\n")
}

func (c *WithComments) setComment(v ...string) {
	c.string = strings.Join(v, "\n")
}

// # Supports:
//   - appdef.ICommenter
type CommentBuilder struct {
	c *WithComments
}

func MakeCommentBuilder(c *WithComments) CommentBuilder {
	return CommentBuilder{c}
}

func (cb *CommentBuilder) SetComment(v ...string) {
	cb.c.setComment(v...)
}

func SetComment(c *WithComments, v ...string) {
	c.setComment(v...)
}
