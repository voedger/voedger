# Driver for AWS DynamoDB storage.

## Overview

This package provides a driver for AWS DynamoDB storage as implementation of interfaces `istorage.IAppStorage` and `istorage.IAppStorageFactory`.


## Configuration

To run DynamoDB locally, use the following command:

```bash
docker run -p 8000:8000 -e AWS_REGION=eu-west-1 -e AWS_ACCESS_KEY_ID=local -e AWS_SECRET_ACCESS_KEY=local amazon/dynamodb-local
```

To connect to the local instance of DynamoDB in go code configure the session as follows:

```go
params := DynamoDBParams{
    EndpointURL:     "http://127.0.0.1:8000", // or your endpoint
    Region:          "eu-west-1", // or your region
    AccessKeyID:     "local", // or your access key
    SecretAccessKey: "local", // or your secret key
    SessionToken:    "",
}
```

## KeySpace

AWS DynamoDB does not have a concept of KeySpace. Keyspaces are emulated using tables with names `<keyspaceName>.values`

## Notes

Partition key and sort key attributes of base tables continue to require non-empty values for all data types, including String and Binary. To make a workaround `byte(0)` is prefixed to each sort key value. See `prefixZero()` and `unprefixZero() functions.

