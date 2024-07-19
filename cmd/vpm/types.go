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
	ModulePath string
	Output     string
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
	Items []interface{}
}

type ormPackageItem struct {
	Package      ormPackageInfo
	WsName       string
	WsDescriptor string
	PkgPath      string
	QName        string
	Name         string
	Type         string
	// TODO: find a way to extract sql-code of the package item from parser/appdef
	SqlContent string
}

type ormTableItem struct {
	ormPackageItem
	Fields       []ormField
	Containers   []ormField
	Keys         []ormField
	NonKeyFields []ormField
}

type ormCommand struct {
	ormPackageItem
	ArgumentObject         interface{}
	UnloggedArgumentObject interface{}
	ResultObjectFields     []ormField
}

func getQName(obj interface{}) string {
	switch t := obj.(type) {
	case ormPackageItem:
		return t.QName
	case ormTableItem:
		return t.QName
	case ormCommand:
		return t.QName
	default:
		panic("unknown type")
	}
}

type ormField struct {
	Table         ormTableItem
	Type          string
	Name          string
	GetMethodName string
	SetMethodName string
}

func hasCommands(p ormPackage) bool {
	for _, item := range p.Items {
		if _, ok := item.(ormCommand); ok {
			return true
		}
	}
	return false
}
