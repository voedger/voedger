/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	airconsts "github.com/untillpro/airs-bp3/packages/air/consts"
	"github.com/voedger/voedger/pkg/appdef"
)

// ///////////////////////////////////
//
//	VIEW: sys.Collection
const (
	Field_PartKey          = "PartKey"
	Field_DocQName         = "DocQName"
	field_DocID            = "DocID"
	field_ElementID        = "ElementID"
	Field_Record           = "Record"
	PartitionKeyCollection = 1 // Always put the BO in the fixed partition
)

var (
	// TODO: air.CollectionView must be air.CollectionView. But if so many records in the view will not be found by air.CollectionView
	// so keep air.CollectionView for backward compatibility
	QNameViewCollection = appdef.NewQName(airconsts.AirPackage, "CollectionView")
)

// ///////////////////////////////////
//
//	FUNC: sys.Collection
const (
	field_Schema = "Schema"
	field_ID     = "ID"
)

var qNameQueryCollection = appdef.NewQName(appdef.SysPackage, "Collection")

// ///////////////////////////////////
//
//	FUNC: air.state
const (
	field_State = "State"
	field_After = "After"
)

var qNameQueryState = appdef.NewQName(appdef.SysPackage, "State")

// ///////////////////////////////////
//
//	FUNC: air.cdoc
const (
	field_xrefs = "xrefs"
)

var (
	qNameCDocFunc = appdef.NewQName(appdef.SysPackage, "CDoc")
)

const DEC = 10
