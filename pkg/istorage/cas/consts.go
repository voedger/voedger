/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package cas

import (
	"time"

	"github.com/gocql/gocql"
)

// ConnectionTimeout s.e.
const (
	initialConnectionTimeout = 30 * time.Second
	ConnectionTimeout        = 30 * time.Second
	retryAttempt             = 3
	SimpleWithReplication    = "{'class': 'SimpleStrategy', 'replication_factor': '1'}"
	DefaultCassandraPort     = 9042
)

var (
	DefaultConsistency = gocql.Quorum
	DefaultCasParams   = CassandraParamsType{
		Hosts:                   "127.0.0.1",
		Port:                    DefaultCassandraPort,
		KeyspaceWithReplication: SimpleWithReplication,
	}
)
