/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package cas

import "time"

// ConnectionTimeout s.e.
const (
	initialConnectionTimeout = 30 * time.Second
	ConnectionTimeout        = 30 * time.Second
	retryAttempt             = 3
	SimpleWithReplication    = "{'class': 'SimpleStrategy', 'replication_factor': '1'}"
)
