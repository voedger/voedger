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
type ormPackageData struct {
	Name              string
	FullPath          string
	HeaderFileContent string
	Imports           []string
	Items             []ormTableData
}

type ormTableData struct {
	Package      ormPackageData
	TypeQName    string
	Name         string
	Type         string
	SqlContent   string
	Fields       []ormFieldData
	Keys         []ormFieldData
	NonKeyFields []ormFieldData
}

type ormFieldData struct {
	Table         ormTableData
	Type          string
	Name          string
	GetMethodName string
}
