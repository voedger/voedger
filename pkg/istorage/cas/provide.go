/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package cas

import (
	"errors"
	"html"

	"github.com/voedger/voedger/pkg/istorage"
)

func Provide(casPar CassandraParamsType) (asf istorage.IAppStorageFactory, err error) {
	if len(casPar.KeyspaceWithReplication) == 0 {
		return nil, errors.New("casPar.KeyspaceWithReplication can not be empty")
	}
	casPar.KeyspaceWithReplication = html.UnescapeString(casPar.KeyspaceWithReplication) // https://dev.untill.com/projects/#!643010
	factory := newCasStorageFactory(casPar)
	return factory, nil
}
