/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

func ProvideRowsProcessorFactory() RowsProcessorFactory {
	return implRowsProcessorFactory
}

func ProvideServiceFactory() ServiceFactory {
	return implServiceFactory
}
