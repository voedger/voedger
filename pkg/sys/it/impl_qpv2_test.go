/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"testing"

	"github.com/voedger/voedger/pkg/vit"
)

func TestQueryProcessor_V2(t *testing.T) {
	t.Skip()
	vit := vit.NewVIT(t, &vit.SharedConfig_App1)
	defer vit.TearDown()

	vit.IFederation.Func("api/v2/users/test1/apps/app1/workspaces/123/docs/app1pkg.category", "{}")
}
