/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package cas

type CassandraParamsType struct {
	// Comma separated list of hosts
	Hosts        string
	Port         int
	Username     string
	Pwd          string
	ProtoVersion int
	CQLVersion   string
	NumRetries   int
	DC           string

	// e.g. "{ 'class' : 'SimpleStrategy', 'replication_factor' : 1 }"
	KeyspaceWithReplication string
}

func (p CassandraParamsType) cqlVersion() string {
	if p.CQLVersion == "" {
		return "3.0.0"
	}
	return p.CQLVersion
}

