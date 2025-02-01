/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package amazondb

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/voedger/voedger/pkg/coreutils"
)

type DynamoDBParams struct {
	EndpointURL     string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type implIAppStorageFactory struct {
	params DynamoDBParams
	iTime  coreutils.ITime
}

type implIAppStorage struct {
	client   *dynamodb.Client
	keySpace string
	iTime    coreutils.ITime
}
