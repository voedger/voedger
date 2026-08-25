/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

// Package featureconformance validates traceability between a Gherkin feature,
// its technical design, and Go feature and technical test sources.
package featureconformance

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Config identifies the sources participating in a feature conformance check.
// Paths are interpreted relative to the package directory from which go test is
// running unless they are absolute.
type Config struct {
	FeatureName                     string
	FeaturePath                     string
	TechnicalDesignPath             string
	FeatureTestPaths                []string
	TechnicalTestPaths              []string
	RequireScenarioOnlyFeatureTests bool
}

type scenarioIdentity struct {
	rule string
	name string
	line int
}

type ruleIdentity struct {
	name string
	line int
}

type featureSpec struct {
	rules     []ruleIdentity
	scenarios []featureScenario
}

type technicalDesignSpec struct {
	rules     []ruleIdentity
	scenarios []scenarioIdentity
}

type featureScenario struct {
	scenarioIdentity
	outline  bool
	steps    []featureStep
	examples []exampleTable
}

type featureStep struct {
	text         string
	line         int
	placeholders []string
}

type exampleTable struct {
	header  string
	columns []string
	line    int
	rows    []exampleRow
}

type exampleRow struct {
	text   string
	values map[string]string
	line   int
}

type goSubtest struct {
	path                   string
	line                   int
	name                   string
	literal                bool
	mayContainScenarioName bool
	comments               []string
}

type scenarioCase struct {
	goSubtest
	identity string
}

type exampleRowRef struct {
	tableIndex int
	rowIndex   int
	table      exampleTable
	row        exampleRow
}

type sourceIdentity struct {
	key         string
	description string
	line        int
}

const bindingResolutionPassLimit = 4

var placeholderPattern = regexp.MustCompile(`<([^<>]+)>`)

// Test reports every conformance diagnostic through t.
func Test(t *testing.T, cfg Config) {
	t.Helper()
	for _, diagnostic := range validate(cfg) {
		t.Error(diagnostic)
	}
}

func validate(cfg Config) []error {
	diagnostics := validateConfig(cfg)
	if len(diagnostics) > 0 {
		return diagnostics
	}

	feature, err := parseFeature(cfg.FeaturePath)
	if err != nil {
		return []error{err}
	}
	technicalDesign, err := parseTechnicalDesign(cfg.TechnicalDesignPath)
	if err != nil {
		return []error{err}
	}

	diagnostics = append(diagnostics, validateTechnicalDesign(cfg, feature, technicalDesign)...)
	diagnostics = append(diagnostics, validateGoSources(cfg, feature)...)
	return diagnostics
}

func validateConfig(cfg Config) []error {
	var diagnostics []error
	if strings.TrimSpace(cfg.FeatureName) == "" {
		diagnostics = append(diagnostics, fmt.Errorf("featureconformance: feature name is empty"))
	}
	if strings.TrimSpace(cfg.FeaturePath) == "" {
		diagnostics = append(diagnostics, fmt.Errorf("featureconformance: feature path is empty"))
	}
	if strings.TrimSpace(cfg.TechnicalDesignPath) == "" {
		diagnostics = append(diagnostics, fmt.Errorf("featureconformance: technical design path is empty"))
	}
	if len(cfg.FeatureTestPaths) == 0 {
		diagnostics = append(diagnostics, fmt.Errorf("featureconformance: feature test paths are empty"))
	}
	return diagnostics
}

func parseFeature(path string) (featureSpec, error) {
	file, err := os.Open(path)
	if err != nil {
		return featureSpec{}, fmt.Errorf("%s: read feature: %w", path, err)
	}
	defer file.Close()

	var spec featureSpec
	currentRule := ""
	currentScenario := -1
	currentExamples := -1
	inDocString := false

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if lineNumber == 1 {
			trimmed = strings.TrimPrefix(trimmed, "\uFEFF")
		}
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, "```") {
			inDocString = !inDocString
			continue
		}
		if inDocString || trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "@") {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "Rule:"):
			currentRule = strings.TrimSpace(strings.TrimPrefix(trimmed, "Rule:"))
			spec.rules = append(spec.rules, ruleIdentity{name: currentRule, line: lineNumber})
			currentScenario = -1
			currentExamples = -1
		case strings.HasPrefix(trimmed, "Scenario Outline:"):
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "Scenario Outline:"))
			spec.scenarios = append(spec.scenarios, featureScenario{
				scenarioIdentity: scenarioIdentity{rule: currentRule, name: name, line: lineNumber},
				outline:          true,
			})
			currentScenario = len(spec.scenarios) - 1
			currentExamples = -1
		case strings.HasPrefix(trimmed, "Scenario:"):
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "Scenario:"))
			spec.scenarios = append(spec.scenarios, featureScenario{
				scenarioIdentity: scenarioIdentity{rule: currentRule, name: name, line: lineNumber},
			})
			currentScenario = len(spec.scenarios) - 1
			currentExamples = -1
		case currentScenario >= 0 && strings.HasPrefix(trimmed, "Examples:"):
			scenario := &spec.scenarios[currentScenario]
			scenario.examples = append(scenario.examples, exampleTable{line: lineNumber})
			currentExamples = len(scenario.examples) - 1
		case currentScenario >= 0 && currentExamples >= 0 && strings.HasPrefix(trimmed, "|"):
			table := &spec.scenarios[currentScenario].examples[currentExamples]
			cells, parseErr := parseTableCells(trimmed)
			if parseErr != nil {
				return featureSpec{}, fmt.Errorf("%s:%d: parse Examples table: %w", path, lineNumber, parseErr)
			}
			if table.header == "" {
				table.header = trimmed
				table.columns = cells
				table.line = lineNumber
				continue
			}
			if len(cells) != len(table.columns) {
				return featureSpec{}, fmt.Errorf("%s:%d: Examples row has %d cells, want %d", path, lineNumber, len(cells), len(table.columns))
			}
			values := make(map[string]string, len(cells))
			for index, column := range table.columns {
				values[column] = cells[index]
			}
			table.rows = append(table.rows, exampleRow{text: trimmed, values: values, line: lineNumber})
		case currentScenario >= 0 && isFeatureStep(trimmed):
			currentExamples = -1
			spec.scenarios[currentScenario].steps = append(spec.scenarios[currentScenario].steps, featureStep{
				text:         trimmed,
				line:         lineNumber,
				placeholders: placeholdersIn(trimmed),
			})
		default:
			if currentExamples >= 0 && !strings.HasPrefix(trimmed, "|") {
				currentExamples = -1
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return featureSpec{}, fmt.Errorf("%s: scan feature: %w", path, err)
	}
	if len(spec.scenarios) == 0 {
		return featureSpec{}, fmt.Errorf("%s: feature has no scenarios", path)
	}
	for _, scenario := range spec.scenarios {
		if scenario.outline && len(allExampleRows(scenario)) == 0 {
			return featureSpec{}, fmt.Errorf("%s:%d: Scenario Outline %q has no Examples rows", path, scenario.line, scenario.name)
		}
	}
	return spec, nil
}

func isFeatureStep(line string) bool {
	for _, keyword := range []string{"Given ", "When ", "Then ", "And ", "But ", "* "} {
		if strings.HasPrefix(line, keyword) {
			return true
		}
	}
	return false
}

func placeholdersIn(text string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(text, -1)
	seen := make(map[string]struct{}, len(matches))
	placeholders := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, exists := seen[match[1]]; exists {
			continue
		}
		seen[match[1]] = struct{}{}
		placeholders = append(placeholders, match[1])
	}
	return placeholders
}

func parseTableCells(row string) ([]string, error) {
	if len(row) < 2 || row[0] != '|' || row[len(row)-1] != '|' {
		return nil, fmt.Errorf("row %q must start and end with a pipe", row)
	}
	parts := strings.Split(row[1:len(row)-1], "|")
	cells := make([]string, len(parts))
	for index, part := range parts {
		cells[index] = strings.TrimSpace(part)
	}
	return cells, nil
}

func parseTechnicalDesign(path string) (technicalDesignSpec, error) {
	file, err := os.Open(path)
	if err != nil {
		return technicalDesignSpec{}, fmt.Errorf("%s: read technical design: %w", path, err)
	}
	defer file.Close()

	var spec technicalDesignSpec
	inScenarios := false
	inFence := false
	currentRule := ""
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		trimmed := strings.TrimSpace(strings.TrimSuffix(scanner.Text(), "\r"))
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if !inScenarios {
			if trimmed == "## Scenarios" {
				inScenarios = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			break
		}
		if strings.HasPrefix(trimmed, "#### ") {
			spec.scenarios = append(spec.scenarios, scenarioIdentity{
				rule: currentRule,
				name: strings.TrimSpace(strings.TrimPrefix(trimmed, "#### ")),
				line: lineNumber,
			})
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			currentRule = strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			spec.rules = append(spec.rules, ruleIdentity{name: currentRule, line: lineNumber})
		}
	}
	if err := scanner.Err(); err != nil {
		return technicalDesignSpec{}, fmt.Errorf("%s: scan technical design: %w", path, err)
	}
	return spec, nil
}

func validateTechnicalDesign(cfg Config, feature featureSpec, technicalDesign technicalDesignSpec) []error {
	diagnostics := validateTechnicalDesignRules(cfg, feature.rules, technicalDesign.rules)
	featureScenarioIdentities := featureIdentities(feature)
	featureCounts := identityCounts(featureScenarioIdentities)
	technicalCounts := identityCounts(technicalDesign.scenarios)

	diagnostics = append(diagnostics, duplicateIdentityDiagnostics(cfg.TechnicalDesignPath, "technical design", technicalDesign.scenarios)...)
	diagnostics = append(diagnostics, duplicateIdentityDiagnostics(cfg.FeaturePath, "feature", featureScenarioIdentities)...)

	for _, expected := range feature.scenarios {
		key := identityKey(expected.scenarioIdentity)
		if technicalCounts[key] > 0 {
			continue
		}
		if actual, found := findScenarioNameInAnotherGroup(technicalDesign.scenarios, expected.scenarioIdentity); found {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical design scenario %q is in Rule %q; expected Rule %q", cfg.TechnicalDesignPath, actual.line, actual.name, actual.rule, expected.rule))
			continue
		}
		if actual, found := findCaseInsensitiveIdentity(technicalDesign.scenarios, expected.scenarioIdentity); found {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical design identities are case-sensitive; got Rule %q scenario %q, want Rule %q scenario %q", cfg.TechnicalDesignPath, actual.line, actual.rule, actual.name, expected.rule, expected.name))
			continue
		}
		diagnostics = append(diagnostics, fmt.Errorf("%s: technical design is missing Rule %q scenario %q", cfg.TechnicalDesignPath, expected.rule, expected.name))
	}
	for _, actual := range technicalDesign.scenarios {
		if featureCounts[identityKey(actual)] > 0 {
			continue
		}
		_, wrongGroup := findScenarioNameInAnotherGroup(featureScenarioIdentities, actual)
		_, caseDifference := findCaseInsensitiveIdentity(featureScenarioIdentities, actual)
		if wrongGroup || caseDifference {
			continue
		}
		diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical design contains extra Rule %q scenario %q", cfg.TechnicalDesignPath, actual.line, actual.rule, actual.name))
	}

	if sameCounts(featureCounts, technicalCounts) {
		expected := featureIdentities(feature)
		for index := range expected {
			if identityKey(expected[index]) != identityKey(technicalDesign.scenarios[index]) {
				actual := technicalDesign.scenarios[index]
				diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical design scenario order differs at position %d: got Rule %q scenario %q, want Rule %q scenario %q", cfg.TechnicalDesignPath, actual.line, index+1, actual.rule, actual.name, expected[index].rule, expected[index].name))
				break
			}
		}
	}
	return diagnostics
}

func validateTechnicalDesignRules(cfg Config, featureRules, technicalRules []ruleIdentity) []error {
	var diagnostics []error
	featureCounts := ruleCounts(featureRules)
	technicalCounts := ruleCounts(technicalRules)
	diagnostics = append(diagnostics, duplicateRuleDiagnostics(cfg.FeaturePath, "feature", featureRules)...)
	diagnostics = append(diagnostics, duplicateRuleDiagnostics(cfg.TechnicalDesignPath, "technical design", technicalRules)...)

	for _, expected := range featureRules {
		if technicalCounts[expected.name] > 0 {
			continue
		}
		if actual, found := findCaseInsensitiveRule(technicalRules, expected.name); found {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical design Rule identities are case-sensitive; got %q, want %q", cfg.TechnicalDesignPath, actual.line, actual.name, expected.name))
			continue
		}
		diagnostics = append(diagnostics, fmt.Errorf("%s: technical design is missing Rule %q", cfg.TechnicalDesignPath, expected.name))
	}
	for _, actual := range technicalRules {
		if featureCounts[actual.name] > 0 {
			continue
		}
		if _, found := findCaseInsensitiveRule(featureRules, actual.name); found {
			continue
		}
		diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical design contains extra Rule %q", cfg.TechnicalDesignPath, actual.line, actual.name))
	}
	if sameCounts(featureCounts, technicalCounts) {
		for index := range featureRules {
			if featureRules[index].name != technicalRules[index].name {
				diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical design Rule order differs at position %d: got %q, want %q", cfg.TechnicalDesignPath, technicalRules[index].line, index+1, technicalRules[index].name, featureRules[index].name))
				break
			}
		}
	}
	return diagnostics
}

func ruleCounts(rules []ruleIdentity) map[string]int {
	counts := make(map[string]int, len(rules))
	for _, rule := range rules {
		counts[rule.name]++
	}
	return counts
}

func duplicateRuleDiagnostics(path, source string, rules []ruleIdentity) []error {
	identities := make([]sourceIdentity, len(rules))
	for index, rule := range rules {
		identities[index] = sourceIdentity{key: rule.name, description: fmt.Sprintf("Rule %q", rule.name), line: rule.line}
	}
	return duplicateSourceIdentityDiagnostics(path, source, identities)
}

func findCaseInsensitiveRule(rules []ruleIdentity, expected string) (ruleIdentity, bool) {
	for _, rule := range rules {
		if strings.EqualFold(rule.name, expected) {
			return rule, true
		}
	}
	return ruleIdentity{}, false
}

func featureIdentities(feature featureSpec) []scenarioIdentity {
	identities := make([]scenarioIdentity, len(feature.scenarios))
	for index, scenario := range feature.scenarios {
		identities[index] = scenario.scenarioIdentity
	}
	return identities
}

func identityCounts(identities []scenarioIdentity) map[string]int {
	counts := make(map[string]int, len(identities))
	for _, identity := range identities {
		counts[identityKey(identity)]++
	}
	return counts
}

func identityKey(identity scenarioIdentity) string {
	return identity.rule + "\x00" + identity.name
}

func formatIdentity(identity string) string {
	parts := strings.SplitN(identity, "\x00", 2)
	return fmt.Sprintf("Rule %q scenario %q", parts[0], parts[1])
}

func duplicateIdentityDiagnostics(path, source string, identities []scenarioIdentity) []error {
	sourceIdentities := make([]sourceIdentity, len(identities))
	for index, identity := range identities {
		key := identityKey(identity)
		sourceIdentities[index] = sourceIdentity{key: key, description: formatIdentity(key), line: identity.line}
	}
	return duplicateSourceIdentityDiagnostics(path, source, sourceIdentities)
}

func duplicateSourceIdentityDiagnostics(path, source string, identities []sourceIdentity) []error {
	var diagnostics []error
	seen := make(map[string]sourceIdentity, len(identities))
	for _, identity := range identities {
		if previous, exists := seen[identity.key]; exists {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: %s contains duplicate %s; first declared at %s:%d", path, identity.line, source, identity.description, path, previous.line))
			continue
		}
		seen[identity.key] = identity
	}
	return diagnostics
}

func findScenarioNameInAnotherGroup(identities []scenarioIdentity, expected scenarioIdentity) (scenarioIdentity, bool) {
	for _, identity := range identities {
		if identity.name == expected.name && identity.rule != expected.rule {
			return identity, true
		}
	}
	return scenarioIdentity{}, false
}

func findCaseInsensitiveIdentity(identities []scenarioIdentity, expected scenarioIdentity) (scenarioIdentity, bool) {
	for _, identity := range identities {
		if strings.EqualFold(identity.rule, expected.rule) && strings.EqualFold(identity.name, expected.name) {
			return identity, true
		}
	}
	return scenarioIdentity{}, false
}

func sameCounts(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for identity, count := range left {
		if right[identity] != count {
			return false
		}
	}
	return true
}

func validateGoSources(cfg Config, feature featureSpec) []error {
	diagnostics := ambiguousScenarioNameDiagnostics(cfg.FeaturePath, feature)
	prefix := cfg.FeatureName + ": scn: "
	casesByScenario := make([][]scenarioCase, len(feature.scenarios))
	seenCaseIdentities := map[string]goSubtest{}

	for _, path := range cfg.FeatureTestPaths {
		subtests, err := parseGoSubtests(path, prefix)
		if err != nil {
			diagnostics = append(diagnostics, err)
			continue
		}
		for _, subtest := range subtests {
			if !subtest.literal {
				if subtest.mayContainScenarioName {
					diagnostics = append(diagnostics, fmt.Errorf("%s:%d: scenario subtest name must be a string literal", subtest.path, subtest.line))
				} else if cfg.RequireScenarioOnlyFeatureTests {
					diagnostics = append(diagnostics, fmt.Errorf("%s:%d: scenario-only feature test file contains a non-literal or untagged subtest", subtest.path, subtest.line))
				}
				continue
			}
			if !strings.HasPrefix(subtest.name, prefix) {
				if cfg.RequireScenarioOnlyFeatureTests {
					diagnostics = append(diagnostics, fmt.Errorf("%s:%d: scenario-only feature test file contains untagged subtest %q", subtest.path, subtest.line, subtest.name))
				}
				continue
			}

			caseIdentity := strings.TrimPrefix(subtest.name, prefix)
			if previous, exists := seenCaseIdentities[caseIdentity]; exists {
				diagnostics = append(diagnostics, fmt.Errorf("%s:%d: duplicate scenario case identity %q; first declared at %s:%d", subtest.path, subtest.line, subtest.name, previous.path, previous.line))
			} else {
				seenCaseIdentities[caseIdentity] = subtest
			}
			scenarioIndex := matchScenario(feature, caseIdentity)
			if scenarioIndex < 0 {
				diagnostics = append(diagnostics, fmt.Errorf("%s:%d: unknown scenario identity %q", subtest.path, subtest.line, subtest.name))
				continue
			}
			casesByScenario[scenarioIndex] = append(casesByScenario[scenarioIndex], scenarioCase{goSubtest: subtest, identity: caseIdentity})
		}
	}

	for _, path := range cfg.TechnicalTestPaths {
		subtests, err := parseGoSubtests(path, prefix)
		if err != nil {
			diagnostics = append(diagnostics, err)
			continue
		}
		for _, subtest := range subtests {
			if strings.HasPrefix(subtest.name, prefix) || subtest.mayContainScenarioName {
				diagnostics = append(diagnostics, fmt.Errorf("%s:%d: technical test source contains feature scenario tag for %q", subtest.path, subtest.line, cfg.FeatureName))
			}
		}
	}

	for index, scenario := range feature.scenarios {
		cases := casesByScenario[index]
		if len(cases) == 0 {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: missing scenario coverage for %q", cfg.FeaturePath, scenario.line, scenario.name))
			continue
		}
		if scenario.outline {
			diagnostics = append(diagnostics, validateOutlineCases(cfg, scenario, cases)...)
			continue
		}
		for _, scenarioCase := range cases {
			diagnostics = append(diagnostics, validateStepComments(cfg, scenario, scenarioCase, nil)...)
		}
	}
	return diagnostics
}

func ambiguousScenarioNameDiagnostics(path string, feature featureSpec) []error {
	var diagnostics []error
	for index, scenario := range feature.scenarios {
		for previousIndex := 0; previousIndex < index; previousIndex++ {
			previous := feature.scenarios[previousIndex]
			if scenario.name != previous.name && !strings.HasPrefix(scenario.name, previous.name+": ") && !strings.HasPrefix(previous.name, scenario.name+": ") {
				continue
			}
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: scenario name %q is ambiguous with Rule %q scenario %q at %s:%d when used as a scenario-tag prefix", path, scenario.line, scenario.name, previous.rule, previous.name, path, previous.line))
		}
	}
	return diagnostics
}

func matchScenario(feature featureSpec, caseIdentity string) int {
	matchedIndex := -1
	matchedLength := -1
	for index, scenario := range feature.scenarios {
		if caseIdentity != scenario.name && !strings.HasPrefix(caseIdentity, scenario.name+": ") {
			continue
		}
		if len(scenario.name) > matchedLength {
			matchedIndex = index
			matchedLength = len(scenario.name)
		}
	}
	return matchedIndex
}

func validateOutlineCases(cfg Config, scenario featureScenario, cases []scenarioCase) []error {
	var diagnostics []error
	rows := allExampleRows(scenario)
	coverage := make(map[string]int, len(rows))
	for _, scenarioCase := range cases {
		if len(rows) > 1 && scenarioCase.identity == scenario.name {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: Scenario Outline case %q requires a row disambiguator", scenarioCase.path, scenarioCase.line, scenarioCase.name))
		}

		matchedRows := matchingRows(scenarioCase.comments, rows)
		if len(matchedRows) == 0 {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: Scenario Outline %q case %q is missing an exact Examples row comment", scenarioCase.path, scenarioCase.line, scenario.name, scenarioCase.name))
			continue
		}
		for _, row := range matchedRows {
			coverage[exampleRowKey(row)]++
			if !containsComment(scenarioCase.comments, row.table.header) {
				diagnostics = append(diagnostics, fmt.Errorf("%s:%d: Scenario Outline %q case %q is missing the exact Examples header comment %q", scenarioCase.path, scenarioCase.line, scenario.name, scenarioCase.name, row.table.header))
			}
			diagnostics = append(diagnostics, validateStepComments(cfg, scenario, scenarioCase, &row.row)...)
		}
	}

	for _, row := range rows {
		count := coverage[exampleRowKey(row)]
		if count == 0 {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: Scenario Outline %q has no feature-test coverage for Examples row %q", cfg.FeaturePath, row.row.line, scenario.name, row.row.text))
		}
		if count > 1 {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: Scenario Outline %q has duplicate feature-test coverage for Examples row %q", cfg.FeaturePath, row.row.line, scenario.name, row.row.text))
		}
	}
	return diagnostics
}

func allExampleRows(scenario featureScenario) []exampleRowRef {
	var rows []exampleRowRef
	for tableIndex, table := range scenario.examples {
		for rowIndex, row := range table.rows {
			rows = append(rows, exampleRowRef{tableIndex: tableIndex, rowIndex: rowIndex, table: table, row: row})
		}
	}
	return rows
}

func matchingRows(comments []string, rows []exampleRowRef) []exampleRowRef {
	var matches []exampleRowRef
	for _, row := range rows {
		if containsComment(comments, row.row.text) {
			matches = append(matches, row)
		}
	}
	return matches
}

func exampleRowKey(row exampleRowRef) string {
	return strconv.Itoa(row.tableIndex) + ":" + strconv.Itoa(row.rowIndex)
}

func validateStepComments(cfg Config, scenario featureScenario, scenarioCase scenarioCase, row *exampleRow) []error {
	var diagnostics []error
	for _, step := range scenario.steps {
		positions := commentPositions(scenarioCase.comments, step.text)
		if len(positions) == 0 {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: scenario %q is missing verbatim step comment %q from %s:%d", scenarioCase.path, scenarioCase.line, scenario.name, step.text, cfg.FeaturePath, step.line))
			continue
		}
		if row == nil || len(step.placeholders) == 0 {
			continue
		}
		if !hasPlaceholderMappings(scenarioCase.comments, positions, step.placeholders, row.values) {
			diagnostics = append(diagnostics, fmt.Errorf("%s:%d: Scenario Outline %q case %q is missing an exact placeholder mapping after step %q", scenarioCase.path, scenarioCase.line, scenario.name, scenarioCase.name, step.text))
		}
	}
	return diagnostics
}

func commentPositions(comments []string, expected string) []int {
	var positions []int
	for index, comment := range comments {
		if comment == expected {
			positions = append(positions, index)
		}
	}
	return positions
}

func hasPlaceholderMappings(comments []string, stepPositions []int, placeholders []string, values map[string]string) bool {
	for _, stepPosition := range stepPositions {
		matched := true
		for offset, placeholder := range placeholders {
			value, exists := values[placeholder]
			if !exists || stepPosition+offset+1 >= len(comments) || comments[stepPosition+offset+1] != formatPlaceholderMapping(placeholder, value) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func formatPlaceholderMapping(placeholder, value string) string {
	if value == "" {
		return placeholder + " ="
	}
	return placeholder + " = " + value
}

func containsComment(comments []string, expected string) bool {
	return len(commentPositions(comments, expected)) > 0
}

func parseGoSubtests(path, scenarioPrefix string) ([]goSubtest, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: read Go source: %w", path, err)
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("%s: parse Go source: %w", path, err)
	}
	var subtests []goSubtest
	globalStringBindings := collectStringBindings(file, nil, true)
	globalScenarioBindings := collectScenarioBindings(file, scenarioPrefix, globalStringBindings, nil, true)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		stringBindings := collectStringBindings(function.Body, withoutParameterBindings(globalStringBindings, function.Type), false)
		scenarioBindings := collectScenarioBindings(function.Body, scenarioPrefix, stringBindings, withoutParameterBindings(globalScenarioBindings, function.Type), false)
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !isSubtestCall(call) {
				return true
			}
			argument := call.Args[0]
			name, literal := stringLiteral(argument)
			if !literal {
				name, _ = staticString(argument, stringBindings)
			}
			funcLiteral := call.Args[1].(*ast.FuncLit)
			subtests = append(subtests, goSubtest{
				path:                   path,
				line:                   fileSet.Position(argument.Pos()).Line,
				name:                   name,
				literal:                literal,
				mayContainScenarioName: strings.HasPrefix(name, scenarioPrefix) || expressionMayContainScenarioName(argument, scenarioPrefix, stringBindings, scenarioBindings),
				comments:               commentsInRange(file.Comments, funcLiteral.Body.Pos(), funcLiteral.Body.End()),
			})
			return true
		})
	}
	return subtests, nil
}

func isSubtestCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Run" || len(call.Args) < 2 {
		return false
	}
	callback, ok := call.Args[1].(*ast.FuncLit)
	return ok && isTestingCallback(callback)
}

func isTestingCallback(callback *ast.FuncLit) bool {
	if callback.Type.Params == nil || len(callback.Type.Params.List) != 1 {
		return false
	}
	pointer, ok := callback.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := pointer.X.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "T"
}

func collectStringBindings(node ast.Node, initial map[string]string, skipFunctions bool) map[string]string {
	bindings := cloneBindings(initial)
	if initial != nil {
		visitNameAssignments(node, skipFunctions, func(name string, _ ast.Expr) {
			delete(bindings, name)
		})
	}
	resolveNameBindings(node, skipFunctions, func(name string, expression ast.Expr) bool {
		value, ok := staticString(expression, bindings)
		if !ok {
			return false
		}
		previous, exists := bindings[name]
		if exists && previous == value {
			return false
		}
		bindings[name] = value
		return true
	})
	return bindings
}

func collectScenarioBindings(node ast.Node, scenarioPrefix string, stringBindings map[string]string, initial map[string]bool, skipFunctions bool) map[string]bool {
	bindings := cloneBindings(initial)
	if initial != nil {
		visitNameAssignments(node, skipFunctions, func(name string, _ ast.Expr) {
			delete(bindings, name)
		})
	}
	resolveNameBindings(node, skipFunctions, func(name string, expression ast.Expr) bool {
		if bindings[name] || !expressionMayContainScenarioName(expression, scenarioPrefix, stringBindings, bindings) {
			return false
		}
		bindings[name] = true
		return true
	})
	return bindings
}

func resolveNameBindings(node ast.Node, skipFunctions bool, resolve func(string, ast.Expr) bool) {
	for pass := 0; pass < bindingResolutionPassLimit; pass++ {
		changed := false
		visitNameAssignments(node, skipFunctions, func(name string, expression ast.Expr) {
			if resolve(name, expression) {
				changed = true
			}
		})
		if !changed {
			break
		}
	}
}

func visitNameAssignments(root ast.Node, skipFunctions bool, visit func(string, ast.Expr)) {
	ast.Inspect(root, func(node ast.Node) bool {
		if skipFunctions {
			if _, ok := node.(*ast.FuncDecl); ok {
				return false
			}
		}
		switch declaration := node.(type) {
		case *ast.ValueSpec:
			if len(declaration.Names) != len(declaration.Values) {
				return true
			}
			for index, name := range declaration.Names {
				if name.Name != "_" {
					visit(name.Name, declaration.Values[index])
				}
			}
		case *ast.AssignStmt:
			if len(declaration.Lhs) != len(declaration.Rhs) {
				return true
			}
			for index, left := range declaration.Lhs {
				name, ok := left.(*ast.Ident)
				if ok && name.Name != "_" {
					visit(name.Name, declaration.Rhs[index])
				}
			}
		}
		return true
	})
}

func cloneBindings[T any](source map[string]T) map[string]T {
	clone := make(map[string]T, len(source))
	for name, value := range source {
		clone[name] = value
	}
	return clone
}

func withoutParameterBindings[T any](bindings map[string]T, functionType *ast.FuncType) map[string]T {
	clone := cloneBindings(bindings)
	if functionType.Params == nil {
		return clone
	}
	for _, field := range functionType.Params.List {
		for _, name := range field.Names {
			delete(clone, name.Name)
		}
	}
	return clone
}

func staticString(expression ast.Expr, bindings map[string]string) (string, bool) {
	switch value := expression.(type) {
	case *ast.BasicLit:
		return stringLiteral(value)
	case *ast.ParenExpr:
		return staticString(value.X, bindings)
	case *ast.Ident:
		resolved, ok := bindings[value.Name]
		return resolved, ok
	case *ast.BinaryExpr:
		if value.Op != token.ADD {
			return "", false
		}
		left, leftOK := staticString(value.X, bindings)
		right, rightOK := staticString(value.Y, bindings)
		if !leftOK || !rightOK {
			return "", false
		}
		return left + right, true
	default:
		return "", false
	}
}

func stringLiteral(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func expressionMayContainScenarioName(expression ast.Expr, scenarioPrefix string, stringBindings map[string]string, scenarioBindings map[string]bool) bool {
	if value, ok := staticString(expression, stringBindings); ok && strings.HasPrefix(value, scenarioPrefix) {
		return true
	}
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && scenarioBindings[identifier.Name] {
			found = true
			return false
		}
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err == nil && strings.Contains(value, scenarioPrefix) {
			found = true
			return false
		}
		return true
	})
	return found
}

func commentsInRange(groups []*ast.CommentGroup, start, end token.Pos) []string {
	var comments []string
	for _, group := range groups {
		if group.Pos() < start || group.End() > end {
			continue
		}
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//") {
				continue
			}
			text := strings.TrimPrefix(comment.Text, "//")
			text = strings.TrimPrefix(text, " ")
			comments = append(comments, strings.TrimSuffix(text, "\r"))
		}
	}
	return comments
}
