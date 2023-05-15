/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
*
 */

package istructs

import "github.com/voedger/voedger/pkg/appdef"

// Base abstract record
type IRecord interface {
	IRowReader
	QName() appdef.QName
	ID() RecordID

	// NullRecordID for documents
	Parent() RecordID

	// Container is empty for documents
	Container() string
}

type IORecord interface {
	IRecord
}

type IEditableRecord interface {
	IRecord
	IsActive() bool
}

type ICRecord interface {
	IEditableRecord
}

type IGRecord interface {
	IEditableRecord
}
