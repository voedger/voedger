/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * * @author Michael Saigachenko
 */

package queryprocessor

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

// FilterFactory creates IFilter from data
type FilterFactory func(data coreutils.MapObject) (IFilter, error)

type IFilter interface {
	IsMatch(fd FieldsKinds, outputRow IOutputRow) (bool, error)
}

// FieldFactory creates IField from data
type FieldFactory func(data interface{}) (IField, error)

// IField is the common interface for all field types
type IField interface {
	Field() string
}

// IResultField is the field from QueryHandler
type IResultField interface {
	IField
}

// IRefField is the field which references to classifier
type IRefField interface {
	IField
	RefField() string
	Key() string
}

// OrderByFactory creates IOrderBy from data
type OrderByFactory func(data coreutils.MapObject) (IOrderBy, error)

type IOrderBy interface {
	Field() string
	IsDesc() bool
}

type IQueryParams interface {
	Elements() []IElement
	Filters() []IFilter
	OrderBy() []IOrderBy
	StartFrom() int64
	Count() int64
}

// ElementFactory creates IElement from data
type ElementFactory func(data coreutils.MapObject) (IElement, error)

type IElement interface {
	Path() IPath
	ResultFields() []IResultField
	RefFields() []IRefField
	NewOutputRow() IOutputRow
}

type IPath interface {
	IsRoot() bool
	Name() string
	AsArray() []string
}

// IWorkpiece is a workpiece for row processor pipeline
type IWorkpiece interface {
	Object() istructs.IObject
	OutputRow() IOutputRow
	EnrichedRootFieldsKinds() FieldsKinds
	PutEnrichedRootFieldKind(name string, kind appdef.DataKind)
}

// IOutputRow is filled by the row processor operators
type IOutputRow interface {
	Set(alias string, value interface{})
	Value(alias string) interface{}
	Values() []interface{}
}

type IQueryMessage interface {
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	Responder() bus.IResponder
	Body() []byte
	RequestCtx() context.Context
	QName() appdef.QName
	//TODO Denis provide partition
	Partition() istructs.PartitionID
	Host() string
	Token() string
}

type IMetrics interface {
	Increase(metricName string, valueDelta float64)
}

type FieldsKinds map[string]appdef.DataKind
