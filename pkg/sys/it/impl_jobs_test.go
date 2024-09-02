/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"testing"
	"time"

	it "github.com/voedger/voedger/pkg/vit"
)

func TestJobjs_BasicUsage(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	time.Sleep(2 * time.Minute)
}
