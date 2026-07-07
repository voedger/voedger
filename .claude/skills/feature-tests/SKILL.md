---
name: feature-tests
description: Use this skill when authoring, designing or reviewing Go `*_test.go` feature tests that verify Gherkin `.feature` scenarios. Use for translating Feature, Scenario, and Scenario Outline content into Go tests while preserving traceability through subtest names, comments, exact Examples table rows, placeholder mappings, and step-to-code alignment.
user-invocable: false
---

# Go feature tests

- Preserve feature-to-test traceability over compactness.
- Prefer the repository's existing Go test helpers, fixture builders, assertions, and package layout.
- Do not run all tests for large suites, `tests/sys`, or `tests/e2e` unless explicitly requested by the user; run only the package, test function, or subtest relevant to the request.
- Do not add sleeps, polling, global state, or cross-test ordering unless the surrounding test infrastructure already requires it.
- Use `t.Helper()` in helper functions that fail the test.

## Test naming

When a Go test corresponds to a `.feature` scenario, use a top-level Go test function for the feature or feature area, and put the feature scenario identity in a subtest name:

```go
func TestUchange(t *testing.T) {
	t.Run("uchange: scn: Type flag: feat", func(t *testing.T) {
		// test body
	})
}
```

Rules:

- `TestUchange` - top-level Go test name in normal Go style; it may follow an existing local convention.
- `uchange:` - lowercase feature name matching the `.feature` file.
- `scn:` - prefix followed by the Scenario or Scenario Outline name from the `.feature` file.
- Brief disambiguator after the scenario name - distinguish multiple tests from the same scenario; for Scenario Outline rows, identify the row using a distinguishing column value.
- Tests without a matching `.feature` scenario must not use `scn:` in the subtest name.
- Prefer explicit subtests over dense table loops when Scenario Outline rows need byte-for-byte table comments and per-row step mappings.

## Step comments

Use Go `//` comments to map Gherkin steps to the code that realizes them:

- `// <Step>` comments label code lines, not the test as a whole. Each comment quotes one Gherkin step (`Given`, `When`, `Then`, `And`, `But`) verbatim and sits immediately above the line or block that realizes it.
- `Given` labels setup code: helper calls, fixture creation, branch checkout, repository setup, workspace creation, or precondition state.
- `When` labels the action under test: command execution, function invocation, API call, event publication, or state transition.
- `Then` / `And` / `But` label assertion code that verifies the outcome.
- Do not combine, summarize, or paraphrase multiple feature steps into one comment.
- If one line realizes multiple feature steps, stack the separate verbatim feature-step comments immediately above that line.
- Do not add a step comment that merely restates what a Scenario Outline table input already shows. Add the comment only when it labels behavior the code actually performs beyond echoing the row.

## Scenario Outline rows

For Scenario Outline examples, copy the Examples table header and the specific row being verified as exactly two Go comment lines immediately near that case:

```go
// | branch             | type | expected_branch_prefix |
// | the default branch | feat | feat/                  |
```

Rules:

- The two table comment lines must be byte-for-byte copies from the `.feature` file except for the leading `// ` comment marker.
- Preserve leading/trailing pipes, every column-alignment whitespace, and every character so `rg` finds the same row text in both files.
- Never reflow a table row across multiple comment lines, drop the pipes, or rewrite the row in a different layout such as `// row: target -> result`.
- For Scenario Outline step comments, keep the step text byte-for-byte verbatim with its `<placeholders>` intact.
- Immediately below the step comment, add one `// placeholder_name = value` line per placeholder that appears in the step, using the value from the row.
- Do not substitute placeholders inline, and do not add free-form paraphrase lines. The `placeholder_name = value` mapping replaces them.
- When one Go test covers multiple Scenario Outline rows, repeat the two-line table comment and placeholder mappings for each row block.

## Examples

Scenario Outline row with setup, action, and assertion:

```go
func TestUchange(t *testing.T) {
	t.Run("uchange: scn: Type flag: feat", func(t *testing.T) {
		// | branch             | type | expected_branch_prefix |
		// | the default branch | feat | feat/                  |
		// Given Engineer is on <branch>
		// branch = the default branch
		repo := setupGitRepo(t)

		// When Engineer invokes uchange action with --type <type>
		// type = feat
		output := runUchange(t, repo, "--kebab-name", "my-change", "--type", "feat")

		// Then a branch is created with prefix <expected_branch_prefix>
		// expected_branch_prefix = feat/
		if got := currentBranch(t, repo); !strings.HasPrefix(got, "feat/") {
			t.Fatalf("branch = %q, want prefix %q\noutput:\n%s", got, "feat/", output)
		}
	})
}
```

Pure input-to-output function test with no meaningful setup/action step comments:

```go
func TestUPR(t *testing.T) {
	t.Run("upr: scn: PR body link handling: mailto: in paragraph", func(t *testing.T) {
		// | link_target             | link_context      | rendered_link      |
		// | mailto:user@example.com | regular paragraph | the link unchanged |
		// Then pr_body renders the link as <rendered_link>
		// rendered_link = the link unchanged
		got := mdDefangRelativeLink("[mail](mailto:user@example.com)")
		if got != "[mail](mailto:user@example.com)" {
			t.Fatalf("rendered link = %q", got)
		}
	})
}
```

Multiple Scenario Outline rows in one Go test - repeat the table comments and mappings in each row block:

```go
func TestUchange(t *testing.T) {
	t.Run("uchange: scn: Issue URL: branch naming: GitHub issue", func(t *testing.T) {
		// | issue_url                               | change_name | branch_name   |
		// | https://github.com/owner/repo/issues/42 | my-feature  | 42-my-feature |
		output := runAction(t, "--issue-url", "https://github.com/owner/repo/issues/42", "--name", "my-feature")
		// Then Git branch is created with name <branch_name>
		// branch_name = 42-my-feature
		if !strings.Contains(output, "git checkout -b 42-my-feature") {
			t.Fatalf("missing branch command in output:\n%s", output)
		}
	})

	t.Run("uchange: scn: Issue URL: branch naming: Jira issue", func(t *testing.T) {
		// | issue_url                                | change_name | branch_name      |
		// | https://jira.example.com/browse/PROJ-123 | fix-bug     | PROJ-123-fix-bug |
		output := runAction(t, "--issue-url", "https://jira.example.com/browse/PROJ-123", "--name", "fix-bug")
		// Then Git branch is created with name <branch_name>
		// branch_name = PROJ-123-fix-bug
		if !strings.Contains(output, "git checkout -b PROJ-123-fix-bug") {
			t.Fatalf("missing branch command in output:\n%s", output)
		}
	})
}
```
