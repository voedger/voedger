/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package amazondb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istorage"
)

func TestBasicUsage(t *testing.T) {
	casPar := DynamoDBParams{
		EndpointURL:     "http://127.0.0.1:8000",
		Region:          "eu-west-1",
		AccessKeyID:     "local",
		SecretAccessKey: "local",
		SessionToken:    "",
	}
	asf, err := Provide(casPar)
	require.NoError(t, err)
	istorage.TechnologyCompatibilityKit(t, asf)
}
