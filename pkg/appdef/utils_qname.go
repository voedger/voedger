/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Compare two qualified names
func CompareQName(a, b QName) int {
	if a.pkg != b.pkg {
		return strings.Compare(a.pkg, b.pkg)
	}
	return strings.Compare(a.entity, b.entity)
}

// Builds a qualified name from two parts (from package name and from entity name)
func NewQName(pkgName, entityName string) QName {
	return QName{pkg: pkgName, entity: entityName}
}

// Parse a qualified name from string.
//
// # Panics:
//   - if string is not a valid qualified name
func MustParseQName(val string) QName {
	q, err := ParseQName(val)
	if err != nil {
		panic(err)
	}
	return q
}

// Parse a qualified name from string
func ParseQName(val string) (res QName, err error) {
	s1, s2, err := ParseQualifiedName(val, QNameQualifierChar)
	return NewQName(s1, s2), err
}

// Parse a qualified name from string using specified delimiter
func ParseQualifiedName(val, delimiter string) (part1, part2 string, err error) {
	s := strings.Split(val, delimiter)
	if len(s) != 2 {
		return NullName, NullName, fmt.Errorf("%w: %v", ErrInvalidQNameStringRepresentation, val)
	}
	return s[0], s[1], nil
}

// Returns package name
func (qn QName) Pkg() string { return qn.pkg }

// Returns entity name
func (qn QName) Entity() string { return qn.entity }

// Returns QName as string
func (qn QName) String() string { return qn.pkg + QNameQualifierChar + qn.entity }

// JSON marshaling support
func (qn QName) MarshalJSON() ([]byte, error) {
	return json.Marshal(qn.pkg + QNameQualifierChar + qn.entity)
}

// need to marshal map[QName]any
func (qn QName) MarshalText() (text []byte, err error) {
	var js []byte
	if js, err = json.Marshal(qn.pkg + QNameQualifierChar + qn.entity); err == nil {
		var res string
		if res, err = strconv.Unquote(string(js)); err == nil {
			text = []byte(res)
		}
	}
	return text, err
}

// JSON unmarshaling support
func (qn *QName) UnmarshalJSON(text []byte) (err error) {
	*qn = QName{}

	str, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	qn.pkg, qn.entity, err = ParseQualifiedName(str, QNameQualifierChar)
	return err
}

// need unmarshal map[QName]any
// golang json looks on UnmarshalText presence only on unmarshal map[QName]any. UnmarshalJSON() will be used anyway
// but no UnmarshalText -> fail to unmarshal map[QName]any
// see https://github.com/golang/go/issues/29732
func (qn *QName) UnmarshalText(text []byte) (err error) {
	return nil
}

// Returns has qName valid package and entity identifiers and error if not
func ValidQName(qName QName) (bool, error) {
	if qName == NullQName {
		return true, nil
	}
	if ok, err := ValidIdent(qName.Pkg()); !ok {
		return false, err
	}
	if ok, err := ValidIdent(qName.Entity()); !ok {
		return false, err
	}
	return true, nil
}

// Slice of QNames.
//
// Slice is sorted and has no duplicates.
//
// Use QNamesFrom() to create QNames slice from variadic arguments.
// Use Add() to add QNames to slice.
// Use Contains() and Find() to search for QName in slice.
type QNames []QName

// Returns slice of QNames from variadic arguments.
//
// Result slice is sorted and has no duplicates.
func QNamesFrom(n ...QName) QNames {
	qq := QNames{}
	qq.Add(n...)
	return qq
}

// Returns slice of QNames from map keys.
//
// Result slice is sorted and has no duplicates.
func QNamesFromMap[V any, M ~map[QName]V](m M) QNames {
	qq := QNames{}
	for k := range m {
		qq.Add(k)
	}
	return qq
}

// Adds QNames to slice. Duplicate values are ignored. Result slice is sorted.
func (qns *QNames) Add(n ...QName) {
	for _, q := range n {
		if i, ok := qns.Find(q); !ok {
			*qns = slices.Insert(*qns, i, q)
		}
	}
}

// Returns true if slice contains specified QName
func (qns QNames) Contains(n QName) bool {
	_, ok := qns.Find(n)
	return ok
}

// Returns index of QName in slice and true if found.
func (qns QNames) Find(n QName) (int, bool) {
	return slices.BinarySearchFunc(qns, n, CompareQName)
}

// Returns is slice with valid qNames and error if not
func ValidQNames(qName ...QName) (ok bool, err error) {
	for _, q := range qName {
		if ok, e := ValidQName(q); !ok {
			err = errors.Join(err, e)
		}
	}
	return err == nil, err
}

// Compare two full qualified names
func CompareFullQName(a, b FullQName) int {
	if a.pkgPath != b.pkgPath {
		return strings.Compare(a.pkgPath, b.pkgPath)
	}
	return strings.Compare(a.entity, b.entity)
}

// Builds a full qualified name from from package path and entity name
func NewFullQName(pkgPath, entityName string) FullQName {
	return FullQName{pkgPath: pkgPath, entity: entityName}
}

// Parse a full qualified name from string.
//
// # Panics:
//   - if string is not a valid full qualified name
func MustParseFullQName(val string) FullQName {
	fqn, err := ParseFullQName(val)
	if err != nil {
		panic(err)
	}
	return fqn
}

// Parse a qualified name from string
func ParseFullQName(val string) (FullQName, error) {
	s1, s2, err := ParseFullQualifiedName(val)
	return NewFullQName(s1, s2), err
}

func ParseFullQualifiedName(val string) (s1, s2 string, err error) {
	p := strings.LastIndex(val, QNameQualifierChar)
	if p < 0 {
		return NullName, NullName, fmt.Errorf("%w: %v", ErrInvalidQNameStringRepresentation, val)
	}

	return val[:p], val[p+1:], nil
}

// Returns package path
func (fqn FullQName) PkgPath() string { return fqn.pkgPath }

// Returns entity name
func (fqn FullQName) Entity() string { return fqn.entity }

// Returns FullQName as string
func (fqn FullQName) String() string { return fqn.pkgPath + QNameQualifierChar + fqn.entity }

// JSON marshaling support
func (fqn FullQName) MarshalJSON() ([]byte, error) {
	return json.Marshal(fqn.pkgPath + QNameQualifierChar + fqn.entity)
}

// need to marshal map[FullQName]any
func (fqn FullQName) MarshalText() (text []byte, err error) {
	var js []byte
	if js, err = json.Marshal(fqn.pkgPath + QNameQualifierChar + fqn.entity); err == nil {
		var res string
		if res, err = strconv.Unquote(string(js)); err == nil {
			text = []byte(res)
		}
	}
	return text, err
}

// JSON unmarshaling support
func (fqn *FullQName) UnmarshalJSON(text []byte) error {
	*fqn = FullQName{}

	str, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	fqn.pkgPath, fqn.entity, err = ParseFullQualifiedName(str)
	return err
}

// need unmarshal map[FullQName]any
// golang json looks on UnmarshalText presence only on unmarshal map[FullQName]any. UnmarshalJSON() will be used anyway
// but no UnmarshalText -> fail to unmarshal map[FullQName]any
// see https://github.com/golang/go/issues/29732
func (fqn *FullQName) UnmarshalText([]byte) error {
	return nil
}

// Returns has FullQName valid package path and entity identifier and error if not
func ValidFullQName(fqn FullQName) (bool, error) {
	if fqn == NullFullQName {
		return true, nil
	}
	if ok, err := ValidPackagePath(fqn.PkgPath()); !ok {
		return false, err
	}
	if ok, err := ValidIdent(fqn.Entity()); !ok {
		return false, err
	}
	return true, nil
}
