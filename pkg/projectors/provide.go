/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import "github.com/voedger/voedger/pkg/appdef"

func ProvideAsyncActualizerFactory() AsyncActualizerFactory {
	return asyncActualizerFactory
}

func ProvideSyncActualizerFactory() SyncActualizerFactory {
	return syncActualizerFactory
}

func ProvideOffsetsSchema(schemas appdef.SchemaCacheBuilder) {
	provideOffsetsSchemaImpl(schemas)
}

func ProvideViewSchema(schemas appdef.SchemaCacheBuilder, qname appdef.QName, buildFunc BuildViewSchemaFunc) {
	provideViewSchemaImpl(schemas, qname, buildFunc)
}
