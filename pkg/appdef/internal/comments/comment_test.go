/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package comments_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
)

func Test_WithComments(t *testing.T) {
	// check interface compatibility
	c := comments.MakeWithComments()
	var _ appdef.IWithComments = &c

	tests := []struct {
		name string
		c    []string
		text string
	}{
		{
			name: "empty",
			c:    nil,
			text: "",
		},
		{
			name: "single",
			c:    []string{"line1"},
			text: "line1",
		},
		{
			name: "multiple",
			c:    []string{"line1", "line2"},
			text: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comments.MakeWithComments(tt.c...)
			require.Equal(t, tt.text, c.Comment())
			require.Equal(t, tt.c, c.CommentLines())
		})
	}

	t.Run("should be ok to use builder", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := comments.MakeWithComments()
				cb := comments.MakeCommentBuilder(&c)
				cb.SetComment(tt.c...)
				require.Equal(t, tt.text, c.Comment())
				require.Equal(t, tt.c, c.CommentLines())
			})
		}
	})

	t.Run("should be ok to use SetComment", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := comments.MakeWithComments()
				comments.SetComment(&c, tt.c...)
				require.Equal(t, tt.text, c.Comment())
				require.Equal(t, tt.c, c.CommentLines())
			})
		}
	})
}
