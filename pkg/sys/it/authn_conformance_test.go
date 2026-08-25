/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package sys_it

import (
	"path/filepath"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/featureconformance"
)

func TestAuthnConformance(t *testing.T) {
	const featureTestPath = "impl_authn_feature_test.go"
	featureconformance.Test(t, featureconformance.Config{
		FeatureName:                     "authn",
		FeaturePath:                     "../../../uspecs/specs/prod/auth/authn.feature",
		TechnicalDesignPath:             "../../../uspecs/specs/prod/auth/authn--td.md",
		FeatureTestPaths:                []string{featureTestPath},
		TechnicalTestPaths:              authnTechnicalTestPaths(t, featureTestPath),
		RequireScenarioOnlyFeatureTests: true,
	})
}

func authnTechnicalTestPaths(t *testing.T, featureTestPath string) []string {
	t.Helper()
	paths, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	technicalPaths := make([]string, 0, len(paths)-1)
	for _, path := range paths {
		if filepath.Clean(path) != filepath.Clean(featureTestPath) {
			technicalPaths = append(technicalPaths, path)
		}
	}
	return technicalPaths
}
