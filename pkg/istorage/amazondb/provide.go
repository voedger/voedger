/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package amazondb

import "github.com/voedger/voedger/pkg/istorage"

func Provide(params DynamoDBParams) (asf istorage.IAppStorageFactory, err error) {
	return &implIAppStorageFactory{
		params: params,
	}, nil
}
