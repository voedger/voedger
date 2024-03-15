/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

type vpmParams struct {
	Dir        string
	TargetDir  string
	IgnoreFile string
	HeaderFile string
}

// packageFiles is a map of package name to a list of files that belong to the package
type packageFiles map[string][]string

// baselineInfo is a struct that is saved to baseline.json file
type baselineInfo struct {
	BaselinePackageUrl string
	Timestamp          string
	GitCommitHash      string `json:",omitempty"`
}

type ignoreInfo struct {
	Ignore []string `yaml:"Ignore"`
}

// types for ORM generating
type ormPackageInfo struct {
	Name              string
	FullPath          string
	HeaderFileContent string
}

type ormPackage struct {
	ormPackageInfo
	Imports []string
	Items   []interface{}
}

type ormPackageItem struct {
	Package    ormPackageInfo
	TypeQName  string
	Name       string
	Type       string
	SqlContent string
}

type ormTableItem struct {
	ormPackageItem
	Fields       []ormField
	Keys         []ormField
	NonKeyFields []ormField
}

type ormCommand struct {
	ormPackageItem
	ArgumentObject         interface{}
	UnloggedArgumentObject interface{}
	ResultObjectFields     []ormField
}

type ormField struct {
	Table         ormTableItem
	Type          string
	Name          string
	GetMethodName string
	SetMethodName string
}
