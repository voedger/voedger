/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package amazondb

import (
	"testing"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
)

func TestBasicUsage(t *testing.T) {
	if !coreutils.IsDynamoDBStorage() {
		t.Skip()
	}
	asf := Provide(DefaultDynamoDBParams, coreutils.MockTime)
	istorage.TechnologyCompatibilityKit(t, asf)
}
