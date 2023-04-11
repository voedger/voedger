/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import "github.com/untillpro/voedger/pkg/istructs"

const (
	pk  = "_PartitionKey"
	cc  = "_ClusteringColumns"
	key = "_FullKey"
	val = "_Value"
)

// Appends suffix to QName entity name and returns new QName
func suffixedQName(q istructs.QName, suff string) QName {
	return istructs.NewQName(q.Pkg(), q.Entity()+suff)
}

// Returns partition key schema name for specified view
func ViewPartitionKeySchemaName(view QName) QName {
	const suff = "_PartitionKey"
	return suffixedQName(view, suff)
}

// Returns clustering columns schema name for specified view
func ViewClusteringColumsSchemaName(view QName) QName {
	const suff = "_ClusteringColumns"
	return suffixedQName(view, suff)
}

// Returns full key schema name for specified view
func ViewFullKeyColumsSchemaName(view QName) QName {
	const suff = "_FullKey"
	return suffixedQName(view, suff)
}

// Returns value schema name for specified view
func ViewValueSchemaName(view QName) QName {
	const suff = "_Value"
	return suffixedQName(view, suff)
}
