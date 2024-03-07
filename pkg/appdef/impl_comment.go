/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "strings"

// # Implements:
//  - IComment
type comment struct {
	string
}

// Creates and returns new comment.
func makeComment(v ...string) comment {
	return comment{strings.Join(v, "\n")}
}

func (c *comment) Comment() string {
	return c.string
}

func (c *comment) CommentLines() []string {
	if len(c.string) == 0 {
		return []string{}
	}
	return strings.Split(c.string, "\n")
}

func (c *comment) setComment(v ...string) {
	c.string = strings.Join(v, "\n")
}

type commentBuilder struct {
	*comment
}

func makeCommentBuilder(comment *comment) commentBuilder {
	return commentBuilder{
		comment: comment,
	}
}

func (cb *commentBuilder) SetComment(v ...string) {
	cb.comment.setComment(v...)
}

type nullComment struct{}

func (c *nullComment) Comment() string        { return "" }
func (c *nullComment) CommentLines() []string { return []string{} }
