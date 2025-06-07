/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package amazondb

const (
	defaultRCU                = 1
	defaultWCU                = 1
	partitionKeyAttributeName = "p_key"
	sortKeyAttributeName      = "c_col"
	valueAttributeName        = "value"
	expireAtAttributeName     = "expire_at"
)

var DefaultDynamoDBParams = DynamoDBParams{
	EndpointURL:     "http://127.0.0.1:8000",
	Region:          "eu-west-1",
	AccessKeyID:     "local",
	SecretAccessKey: "local",
	SessionToken:    "",
}
