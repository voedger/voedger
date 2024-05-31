/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # QName
//
// Qualified name
//
// <pkg>.<entity>
type QName struct {
	pkg    string
	entity string
}

// Slice of QNames.
//
// Slice is sorted and has no duplicates.
//
// Use QNamesFrom() to create QNames slice from variadic arguments.
// Use Add() to add QNames to slice.
// Use Contains() and Find() to search for QName in slice.
type QNames []QName

// # FullQName
//
// Full qualified name
//
// <pkgPath>.<entity>
type FullQName struct {
	pkgPath string
	entity  string
}
