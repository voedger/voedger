/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package featureconformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validFeatureSource = `Feature: Demo

  Rule: First rule

    Scenario: Plain behavior
      Given plain setup
      When plain action
      Then plain result

  Rule: Second rule

    Scenario Outline: Outlined behavior
      Given input "<input>"
      When outline action
      Then result is "<result>"

      Examples:
        | input | result |
        | alpha | one    |
        | beta  | two    |
`

const validTechnicalDesignSource = `# Feature technical design: Demo

## Scenarios

### First rule

#### Plain behavior

` + "```text" + `
plain flow
` + "```" + `

### Second rule

#### Outlined behavior

` + "```text" + `
outlined flow
` + "```" + `
`

const validFeatureTestSource = `package sample

import "testing"

func TestDemo(t *testing.T) {
	t.Run("demo: scn: Plain behavior", func(t *testing.T) {
		// Given plain setup
		plainSetup()
		// When plain action
		plainAction()
		// Then plain result
		plainResult()
	})

	t.Run("demo: scn: Outlined behavior: alpha", func(t *testing.T) {
		// | input | result |
		// | alpha | one    |
		// Given input "<input>"
		// input = alpha
		outlineSetup("alpha")
		// When outline action
		outlineAction()
		// Then result is "<result>"
		// result = one
		outlineResult("one")
	})

	t.Run("demo: scn: Outlined behavior: beta", func(t *testing.T) {
		// | input | result |
		// | beta  | two    |
		// Given input "<input>"
		// input = beta
		outlineSetup("beta")
		// When outline action
		outlineAction()
		// Then result is "<result>"
		// result = two
		outlineResult("two")
	})
}
`

func TestConformanceAcceptsExactSources(t *testing.T) {
	cfg := writeConformanceFixture(t, validFeatureSource, validTechnicalDesignSource, validFeatureTestSource, "package sample\n")
	Test(t, cfg)
}

func TestConformanceRejectsTechnicalDesignDrift(t *testing.T) {
	tests := []struct {
		name   string
		td     string
		needle string
	}{
		{
			name:   "missing scenario",
			td:     strings.Replace(validTechnicalDesignSource, "#### Plain behavior\n", "", 1),
			needle: "technical design",
		},
		{
			name:   "extra scenario",
			td:     strings.Replace(validTechnicalDesignSource, "#### Plain behavior\n", "#### Plain behavior\n\n#### Extra behavior\n", 1),
			needle: "technical design",
		},
		{
			name: "reordered scenarios",
			td: `## Scenarios

### Second rule
#### Outlined behavior
` + "```text\nflow\n```" + `

### First rule
#### Plain behavior
` + "```text\nflow\n```" + `
`,
			needle: "order",
		},
		{
			name: "mis-grouped scenario",
			td: `## Scenarios

### First rule

### Second rule
#### Plain behavior
` + "```text\nflow\n```" + `
#### Outlined behavior
` + "```text\nflow\n```" + `
`,
			needle: "group",
		},
		{
			name:   "case difference",
			td:     strings.Replace(validTechnicalDesignSource, "#### Plain behavior", "#### Plain Behavior", 1),
			needle: "case-sensitive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := writeConformanceFixture(t, validFeatureSource, tc.td, validFeatureTestSource, "package sample\n")
			requireConformanceDiagnostic(t, validate(cfg), tc.needle)
		})
	}
}

func TestConformanceRejectsInvalidScenarioIdentities(t *testing.T) {
	tests := []struct {
		name   string
		source string
		needle string
	}{
		{
			name:   "missing coverage",
			source: strings.Replace(validFeatureTestSource, "demo: scn: Plain behavior", "demo: technical: Plain behavior", 1),
			needle: "missing scenario coverage",
		},
		{
			name:   "unknown scenario",
			source: strings.Replace(validFeatureTestSource, "demo: scn: Plain behavior", "demo: scn: Unknown behavior", 1),
			needle: "unknown scenario",
		},
		{
			name:   "duplicate case identity",
			source: strings.Replace(validFeatureTestSource, "demo: scn: Outlined behavior: beta", "demo: scn: Outlined behavior: alpha", 1),
			needle: "duplicate",
		},
		{
			name: "ambiguous outline cases",
			source: strings.NewReplacer(
				"demo: scn: Outlined behavior: alpha", "demo: scn: Outlined behavior",
				"demo: scn: Outlined behavior: beta", "demo: scn: Outlined behavior",
			).Replace(validFeatureTestSource),
			needle: "disambiguator",
		},
		{
			name: "dynamic scenario name",
			source: strings.NewReplacer(
				"func TestDemo(t *testing.T) {", "func TestDemo(t *testing.T) {\n\tplainName := \"demo: scn: Plain behavior\"",
				"t.Run(\"demo: scn: Plain behavior\"", "t.Run(plainName",
			).Replace(validFeatureTestSource),
			needle: "literal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := writeConformanceFixture(t, validFeatureSource, validTechnicalDesignSource, tc.source, "package sample\n")
			requireConformanceDiagnostic(t, validate(cfg), tc.needle)
		})
	}
}

func TestConformanceRejectsTraceabilityCommentDrift(t *testing.T) {
	tests := []struct {
		name   string
		source string
		needle string
	}{
		{
			name:   "non-verbatim step",
			source: strings.Replace(validFeatureTestSource, "// Given plain setup", "// Given a plain setup", 1),
			needle: "verbatim step",
		},
		{
			name:   "outline header spacing",
			source: strings.Replace(validFeatureTestSource, "// | input | result |", "// | input  | result |", 1),
			needle: "Examples header",
		},
		{
			name:   "outline row spacing",
			source: strings.Replace(validFeatureTestSource, "// | beta  | two    |", "// | beta | two |", 1),
			needle: "Examples row",
		},
		{
			name:   "missing placeholder mapping",
			source: strings.Replace(validFeatureTestSource, "\t\t// result = two\n", "", 1),
			needle: "placeholder mapping",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := writeConformanceFixture(t, validFeatureSource, validTechnicalDesignSource, tc.source, "package sample\n")
			requireConformanceDiagnostic(t, validate(cfg), tc.needle)
		})
	}
}

func TestConformanceRejectsStrictFeatureFileAndTechnicalTagViolations(t *testing.T) {
	t.Run("untagged feature subtest", func(t *testing.T) {
		source := strings.Replace(validFeatureTestSource, "}\n", `
	t.Run("technical regression", func(t *testing.T) {})
}
`, 1)
		cfg := writeConformanceFixture(t, validFeatureSource, validTechnicalDesignSource, source, "package sample\n")
		requireConformanceDiagnostic(t, validate(cfg), "scenario-only")
	})

	t.Run("scenario tag in technical source", func(t *testing.T) {
		technicalSource := `package sample

import "testing"

func TestTechnical(t *testing.T) {
	t.Run("demo: scn: Plain behavior", func(t *testing.T) {})
}
`
		cfg := writeConformanceFixture(t, validFeatureSource, validTechnicalDesignSource, validFeatureTestSource, technicalSource)
		requireConformanceDiagnostic(t, validate(cfg), "technical test")
	})
}

func TestConformanceReadsSourcesWithoutCompilingOrExecutingThem(t *testing.T) {
	technicalSource := `package sample

func unresolvedTechnicalTestSource() {
	missingSymbol()
}
`
	cfg := writeConformanceFixture(t, validFeatureSource, validTechnicalDesignSource, validFeatureTestSource, technicalSource)
	Test(t, cfg)
}

func writeConformanceFixture(t *testing.T, featureSource, technicalDesignSource, featureTestSource, technicalTestSource string) Config {
	t.Helper()
	dir := t.TempDir()
	featurePath := writeConformanceFile(t, dir, "demo.feature", featureSource)
	technicalDesignPath := writeConformanceFile(t, dir, "demo--td.md", technicalDesignSource)
	featureTestPath := writeConformanceFile(t, dir, "demo_feature_test.go", featureTestSource)
	technicalTestPath := writeConformanceFile(t, dir, "demo_test.go", technicalTestSource)
	return Config{
		FeatureName:                     "demo",
		FeaturePath:                     featurePath,
		TechnicalDesignPath:             technicalDesignPath,
		FeatureTestPaths:                []string{featureTestPath},
		TechnicalTestPaths:              []string{technicalTestPath},
		RequireScenarioOnlyFeatureTests: true,
	}
}

func writeConformanceFile(t *testing.T, dir, name, source string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func requireConformanceDiagnostic(t *testing.T, diagnostics []error, needle string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Error(), needle) {
			return
		}
	}
	t.Fatalf("diagnostics %q do not contain %q", diagnostics, needle)
}
