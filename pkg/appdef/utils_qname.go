/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
)

//—————————————————————————————
//— QName —————————————————————
//—————————————————————————————

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
		return NullName, NullName, ErrConvert("string «%s» to qualified name", val)
	}
	return s[0], s[1], nil
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

//—————————————————————————————
//—  QNames ———————————————————
//—————————————————————————————

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

// Parse a qualified names from strings.
//
// # Panics:
//   - if strings contains not a valid qualified name
func MustParseQNames(val ...string) QNames {
	q := QNames{}
	for _, v := range val {
		q.Add(MustParseQName(v))
	}
	return q
}

// Parse a qualified name from string
func ParseQNames(val ...string) (res QNames, err error) {
	res = QNames{}
	for _, v := range val {
		q, e := ParseQName(v)
		if e == nil {
			res.Add(q)
		} else {
			err = errors.Join(err, e)
		}
	}
	if err == nil {
		return res, nil
	}
	return nil, err
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

// Adds QNames to slice. Duplicate values are ignored. Result slice is sorted.
func (qns *QNames) Add(n ...QName) {
	for _, q := range n {
		if i, ok := qns.Find(q); !ok {
			*qns = slices.Insert(*qns, i, q)
		}
	}
}

// Collect QNames using iterator. Duplicate values are ignored. Result slice is sorted.
func (qns *QNames) Collect(seq func(func(QName) bool)) {
	for n := range seq {
		qns.Add(n)
	}
}

// Returns true if slice contains specified QName
func (qns QNames) Contains(n QName) bool {
	_, ok := qns.Find(n)
	return ok
}

// Returns true if slice contains all specified QNames.
//
// If no names specified then returns true.
func (qns QNames) ContainsAll(names ...QName) bool {
	for _, n := range names {
		if !qns.Contains(n) {
			return false
		}
	}
	return true
}

// Returns true if slice contains any from specified QName.
//
// If no names specified then returns true.
func (qns QNames) ContainsAny(names ...QName) bool {
	for _, n := range names {
		if qns.Contains(n) {
			return true
		}
	}
	return len(names) == 0
}

// Returns index of QName in slice and true if found.
func (qns QNames) Find(n QName) (int, bool) {
	return slices.BinarySearchFunc(qns, n, CompareQName)
}

//—————————————————————————————
//— FullQName —————————————————
//—————————————————————————————

// Compare two full qualified names
func CompareFullQName(a, b FullQName) int {
	if a.pkgPath != b.pkgPath {
		return strings.Compare(a.pkgPath, b.pkgPath)
	}
	return strings.Compare(a.entity, b.entity)
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

// Builds a full qualified name from from package path and entity name
func NewFullQName(pkgPath, entityName string) FullQName {
	return FullQName{pkgPath: pkgPath, entity: entityName}
}

// Parse a qualified name from string. Result is FullQName or error
func ParseFullQName(val string) (FullQName, error) {
	s1, s2, err := ParseFullQualifiedName(val)
	return NewFullQName(s1, s2), err
}

// Parse a qualified name from string. Result is package path and local name or error
func ParseFullQualifiedName(val string) (p, n string, err error) {
	i := strings.LastIndex(val, QNameQualifierChar)
	if i < 0 {
		return NullName, NullName, ErrConvert("string «%s» to QName", val)
	}

	return val[:i], val[i+1:], nil
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

// Returns package path
func (fqn FullQName) PkgPath() string { return fqn.pkgPath }

// Returns entity local name
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

//—————————————————————————————
//— AppQName ——————————————————
//—————————————————————————————

// Builds a qualified application name from two parts (from owner name and from local application name)
func NewAppQName(owner, name string) AppQName {
	return AppQName{owner: owner, name: name}
}

// Parse application qualified name from string. Result is AppQName.
//
// # Panics
//   - if string is not a valid application qualified name
func MustParseAppQName(val string) AppQName {
	n, err := ParseAppQName(val)
	if err != nil {
		panic(err)
	}
	return n
}

// Parse application qualified name from string. Result is AppQName or error
func ParseAppQName(val string) (AppQName, error) {
	o, n, err := ParseQualifiedName(val, AppQNameQualifierChar)
	return NewAppQName(o, n), err
}

// Returns owner name
func (aqn AppQName) Owner() string { return aqn.owner }

// Returns application local name
func (aqn AppQName) Name() string { return aqn.name }

// Returns AppQName as string
func (aqn AppQName) String() string { return aqn.owner + AppQNameQualifierChar + aqn.name }

// Returns true if application is owned by system
func (aqn AppQName) IsSys() bool { return aqn.owner == SysOwner }

func (aqn AppQName) MarshalJSON() ([]byte, error) {
	return json.Marshal(aqn.owner + AppQNameQualifierChar + aqn.name)
}

// need to marshal map[AppQName]any
func (aqn AppQName) MarshalText() (text []byte, err error) {
	var js []byte
	if js, err = json.Marshal(aqn.owner + AppQNameQualifierChar + aqn.name); err == nil {
		var res string
		if res, err = strconv.Unquote(string(js)); err == nil {
			text = []byte(res)
		}
	}
	return text, err
}

func (aqn *AppQName) UnmarshalJSON(text []byte) (err error) {
	*aqn = AppQName{}
	str, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	aqn.owner, aqn.name, err = ParseQualifiedName(str, AppQNameQualifierChar)
	return err
}

// need to unmarshal map[AppQName]any
// golang json looks on UnmarshalText presence only on unmarshal map[QName]any. UnmarshalJSON() will be used anyway
// but no UnmarshalText -> fail to unmarshal map[AppQName]any
// see https://github.com/golang/go/issues/29732
func (aqn *AppQName) UnmarshalText(text []byte) (err error) {
	// notest
	return nil
}
