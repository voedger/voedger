/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"github.com/voedger/voedger/pkg/schemas"
)

func ProvideAsyncActualizerFactory() AsyncActualizerFactory {
	return asyncActualizerFactory
}

func ProvideSyncActualizerFactory() SyncActualizerFactory {
	return syncActualizerFactory
}

func ProvideOffsetsSchema(schemas schemas.SchemaCacheBuilder) {
	provideOffsetsSchemaImpl(schemas)
}

func ProvideViewSchema(schemas schemas.SchemaCacheBuilder, qname schemas.QName, buildFunc BuildViewSchemaFunc) {
	provideViewSchemaImpl(schemas, qname, buildFunc)
}
