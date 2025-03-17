/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package openapi

import (
	"iter"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/processors/query2"
)

type SchemaMeta struct {
	SchemaTitle   string
	SchemaVersion string
	AppName       appdef.AppQName
}

type PublishedTypesFunc func(ws appdef.IWorkspace, role appdef.QName) iter.Seq2[appdef.IType,
	iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]]

type ischema interface {
	appdef.IType
	appdef.IWithFields
}

type pathItem struct {
	Method  string
	Path    string
	ApiPath query2.ApiPath
}
