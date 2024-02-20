# Driver for AWS DynamoDB storage.

## Overview

This package provides a driver for AWS DynamoDB storage as implementation of interfaces `istorage.IAppStorage` and `istorage.IAppStorageFactory`. 


## Configuration

To run DynamoDB locally, use the following command:

```bash
docker run -p 8000:8000 -e AWS_REGION={AWS_REGION} -e AWS_ACCESS_KEY_ID={ACESS_KEY_ID} -e AWS_SECRET_ACCESS_KEY={SECRET_ACCESS_KEY} amazon/dynamodb-local
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

AWS DynamoDB does not have a concept of KeySpace. Instead, it has tables. The table name is used as a KeySpace.

## Notes

Partition key and sort key attributes of base tables continue to require non-empty values for all data types, including String and Binary. That is why there is a workaround for null values for clustering columns (see prefixZero and unprefixZero functions).
