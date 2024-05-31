/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author: Maxim Geraskin
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

// AppQName is unique in cluster federation
// <owner>/<name>
// sys/registry
// unTill/airs-bp
// test1/app1
// test1/app2
// test2/app1
// test2/app2
type AppQName struct {
	owner, name string
}
