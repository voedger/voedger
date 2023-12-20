/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorageimpl/istoragecas"
)

const (
	EPSchemasFS             extensionpoints.EPKey = "SchemasFS"
	casStorageTypeCas1                            = "cas1"
	casStorageTypeCas3                            = "cas3"
	cas1ReplicationStrategy                       = "{'class': 'SimpleStrategy', 'replication_factor': '1'}"
	cas3ReplicationStrategy                       = "{ 'class': 'NetworkTopologyStrategy', 'dc1': 2, 'dc2': 1}"
)

var defaultCasParams = istoragecas.CassandraParamsType{
	Hosts:    "db-node-1,db-node-2,db-node-3",
	Port:     9042,
	Username: "cassandra",
	Pwd:      "cassandra",
}
