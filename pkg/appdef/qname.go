/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	// System package name
	SysPackage = "sys"

	// Used as delimiter in qualified names
	QNameQualifierChar = "."

	// Used as prefix for names of system fields and containers
	SystemPackagePrefix = SysPackage + QNameQualifierChar
)

// Null (empty) QName
var (
	QNameForNull = NewQName(NullName, NullName)
	NullQName    = QNameForNull
)

// Builds a qualfied name from two parts (from pakage name and from entity name)
func NewQName(pkgName, entityName string) QName {
	return QName{pkg: pkgName, entity: entityName}
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
	js, err := json.Marshal(qn.pkg + QNameQualifierChar + qn.entity)
	if err != nil {
		// notest
		return nil, err
	}
	res, err := strconv.Unquote(string(js))
	if err != nil {
		// notest
		return nil, err
	}
	return []byte(res), nil
}

// JSON unmarshaling support
func (qn *QName) UnmarshalJSON(text []byte) (err error) {
	*qn = QName{}

	str, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	qn.pkg, qn.entity, err = ParseQualifiedName(string(str), QNameQualifierChar)
	return err
}

// need unmarshal map[QName]any
// golang json looks on UnmarshalText presence only on unmarshal map[QName]any. UnmarshalJSON() will be used anyway
// but no UnmarshalText -> fail to unmarshal map[QName]any
// see https://github.com/golang/go/issues/29732
func (qn *QName) UnmarshalText(text []byte) (err error) {
	// notest
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
