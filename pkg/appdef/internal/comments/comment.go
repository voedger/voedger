/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package comments

import (
	"iter"
	"slices"
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

func (c *WithComments) CommentLines() iter.Seq[string] {
	if len(c.string) == 0 {
		return func(func(string) bool) {}
	}
	return slices.Values(strings.Split(c.string, "\n"))
}

func (c *WithComments) setComment(v ...string) {
	c.string = strings.Join(v, "\n")
}

// # Supports:
//   - appdef.ICommenter
type CommentBuilder struct {
	*WithComments
}

func MakeCommentBuilder(comment *WithComments) CommentBuilder {
	return CommentBuilder{comment}
}

func (cb *CommentBuilder) SetComment(v ...string) {
	cb.WithComments.setComment(v...)
}

// # Supports
//   - appdef.IWithComments
type NullComment struct{}

func (c *NullComment) Comment() string                { return "" }
func (c *NullComment) CommentLines() iter.Seq[string] { return func(func(string) bool) {} }

func SetComment(c *WithComments, v ...string) {
	c.setComment(v...)
}
