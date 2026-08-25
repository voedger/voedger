# testingu
`testingu` is a Go package designed to facilitate testing of command-line interface (CLI) applications. It provides tools to capture standard output and standard error, and to validate expected outcomes for different argument combinations.

## Installation
To install the `testingu` package, run:

```sh
go get -u github.com/voedger/voedger/pkg/goutils/testingu
```

## Table of Contents

- [MockTime - ITime implementation for testing](mocktime.md)
- [RunCLITests - util for testing CLI](clitest.md)
- [`featureconformance` - source-level feature traceability checks](featureconformance)

## Feature conformance

The `featureconformance` package validates that a Gherkin feature, its technical-design scenario headings, and its Go feature tests stay aligned. It reads source files only: the check does not compile the configured Go sources or initialize their test fixtures.

Add a thin runner beside the feature tests and provide paths relative to that package's test working directory:

```go
func TestExampleConformance(t *testing.T) {
	featureconformance.Test(t, featureconformance.Config{
		FeatureName:         "example",
		FeaturePath:         "../../../uspecs/specs/prod/example/example.feature",
		TechnicalDesignPath: "../../../uspecs/specs/prod/example/example--td.md",
		FeatureTestPaths:    []string{"impl_example_feature_test.go"},
		TechnicalTestPaths:  []string{"example_test.go"},
	})
}
```

Set `RequireScenarioOnlyFeatureTests: true` when every `t.Run` in the configured feature-test sources must use a literal `<feature>: scn: <Scenario>` identity. Configured technical-test sources must not contain that feature's scenario tags.
