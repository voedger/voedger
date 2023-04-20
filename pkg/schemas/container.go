/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"strings"
)

const SystemContainer_ViewPartitionKey = SystemPackagePrefix + "pkey"
const SystemContainer_ViewClusteringCols = SystemPackagePrefix + "ccols"
const SystemContainer_ViewValue = SystemPackagePrefix + "val"

// Implements Container interface
type container struct {
	name      string
	schema    QName
	minOccurs Occurs
	maxOccurs Occurs
}

func newContainer(name string, schema QName, minOccurs, maxOccurs Occurs) container {
	return container{
		name:      name,
		schema:    schema,
		minOccurs: minOccurs,
		maxOccurs: maxOccurs,
	}
}

func (cont *container) IsSys() bool { return IsSysContainer(cont.name) }

func (cont *container) MaxOccurs() Occurs { return cont.maxOccurs }

func (cont *container) MinOccurs() Occurs { return cont.minOccurs }

func (cont *container) Name() string { return cont.name }

func (cont *container) Schema() QName { return cont.schema }

// Returns is container system
func IsSysContainer(n string) bool {
	return strings.HasPrefix(n, SystemPackagePrefix) && // fast check
		// then more accuracy
		((n == SystemContainer_ViewPartitionKey) ||
			(n == SystemContainer_ViewClusteringCols) ||
			(n == SystemContainer_ViewValue))
}
