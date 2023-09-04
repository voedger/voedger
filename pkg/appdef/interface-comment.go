/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// See [Issue #488](https://github.com/voedger/voedger/issues/488)
//
// Any definition may have comment
//
// Ref to commented.go for implementation
type IComment interface {
	// Returns comment
	Comment() string
}

type ICommentBuilder interface {
	// Sets comment as string with lines, concatenated with LF
	SetComment(...string)
}
