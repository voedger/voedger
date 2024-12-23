/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
)

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
	Items []any
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

type ormProjector struct {
	ormPackageItem
	On []ormProjectorEventItem
}

type ormProjectorEventItem struct {
	ormPackageItem
	EventItem      any
	Ops            []appdef.OperationKind
	Projector      ormProjector
	SkipGeneration bool
}

type ormCommand struct {
	ormPackageItem
	ArgumentObject         interface{}
	UnloggedArgumentObject interface{}
	ResultObjectFields     []ormField
}

func getQName(obj interface{}) string {
	switch t := obj.(type) {
	case ormProjector:
		return t.QName
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

func isExecutableWithParam(p ormProjector) bool {
	for _, item := range p.On {
		if doesExecuteWithParam(item) {
			return true
		}
	}

	return false
}

func doesExecuteWithParam(p ormProjectorEventItem) bool {
	return slices.Contains(p.Ops, appdef.OperationKind_ExecuteWithParam)
}

func hasEventItemName(p ormProjector, name string) bool {
	for _, item := range p.On {
		if item.Name == name {
			return true
		}
	}

	return false
}

func doesExecuteOn(p ormProjectorEventItem) bool {
	return slices.Contains(p.Ops, appdef.OperationKind_Execute)
}

func doesTriggerOnCUD(p ormProjectorEventItem) bool {
	//nnv: Is OperationKind_Activate / OperationKind_Deactivate triggers CUD too?
	return slices.Contains(p.Ops, appdef.OperationKind_Insert) || slices.Contains(p.Ops, appdef.OperationKind_Update)
}
