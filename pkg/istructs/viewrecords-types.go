/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
*
 */

package istructs

import "github.com/voedger/voedger/pkg/appdef"

// ref. also https://cassandra.apache.org/doc/latest/cassandra/cql/ddl.html
// FIXME implement IRowWriter
type IKeyBuilder interface {
	IRowWriter
	PartitionKey() IRowWriter
	ClusteringColumns() IRowWriter
	// Equals returns is src key builder has the same QName and field values. See #!21906
	Equals(src IKeyBuilder) bool

	// Puts key to bytes for specified workspace id
	//
	// Returns error if there were errors when calling Put-methods
	ToBytes(WSID) (pk, cc []byte, err error)
}

type IValueBuilder interface {
	IRowWriter

	// @Tricky
	PutRecord(appdef.FieldName, IRecord)

	// @Tricky
	PutEvent(appdef.FieldName, IDbEvent)
	Build() IValue

	// Writes value data to bytes.
	//
	// Returns error if there were errors when calling Put-methods
	ToBytes() ([]byte, error)
}

// @Tricky
type IValue interface {
	IRowReader

	// The following methods panic if cell type does not match

	AsRecord(appdef.FieldName) IRecord
	AsEvent(appdef.FieldName) IDbEvent
}

type IKey interface {
	IRowReader
}
