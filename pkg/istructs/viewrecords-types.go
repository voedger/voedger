/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
*
 */

package istructs

// ref. also https://cassandra.apache.org/doc/latest/cassandra/cql/ddl.html
// FIXME implement IRowWriter
type IKeyBuilder interface {
	IRowWriter
	PartitionKey() IRowWriter
	ClusteringColumns() IRowWriter
	// Equals returns is src key builder has the same QName and field values. See #!21906
	Equals(src IKeyBuilder) bool
}

type IValueBuilder interface {
	IRowWriter

	// @Tricky
	PutRecord(name string, record IRecord)

	// @Tricky
	PutEvent(name string, event IDbEvent)
	Build() IValue
}

// @Tricky
type IValue interface {
	IRowReader

	// The following methods panic if cell type does not match

	AsRecord(name string) (record IRecord)
	AsEvent(name string) (event IDbEvent)
}

type IKey interface {
	IRowReader
}
