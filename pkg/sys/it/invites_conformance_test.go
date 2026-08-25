/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package sys_it

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/featureconformance"
)

func TestInvitesConformance(t *testing.T) {
	featureconformance.Test(t, featureconformance.Config{
		FeatureName:         "invites",
		FeaturePath:         "../../../uspecs/specs/prod/auth/invites.feature",
		TechnicalDesignPath: "../../../uspecs/specs/prod/auth/invites--td.md",
		FeatureTestPaths:    []string{"impl_invites_feature_test.go"},
		TechnicalTestPaths:  []string{"impl_invite_test.go"},
	})
}
