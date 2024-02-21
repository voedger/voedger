/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package amazondb

import (
	"testing"

	"github.com/voedger/voedger/pkg/istorage"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestBasicUsage(t *testing.T) {
	if !coreutils.IsDynamoDBStorage() {
		t.Skip()
	}
	params := DynamoDBParams{
		EndpointURL:     "http://127.0.0.1:8000",
		Region:          "eu-west-1",
		AccessKeyID:     "local",
		SecretAccessKey: "local",
		SessionToken:    "",
	}
	asf := Provide(params)
	istorage.TechnologyCompatibilityKit(t, asf)
}
