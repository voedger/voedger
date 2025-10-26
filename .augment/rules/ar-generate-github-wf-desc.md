---
type: "manual"
---

# Prompt for Generating GitHub Workflow Documentation

## Objective

Generate comprehensive documentation for GitHub Actions workflows that traces execution flow from GitHub events through workflow calls to final outputs. The documentation should include:

1. **README.md** - Detailed execution trace showing all workflows, events, and steps
2. **EXECUTION_FLOW_DIAGRAMS.md** - Visual diagrams showing execution and data flow

## README.md Requirements

### Structure

- **Header**: "GitHub Workflow Execution Trace"
- **Sections**: One section per GitHub event that triggers workflows
- **Format**: Markdown with links to source files and line numbers

### Content for Each Event Section

For each GitHub event, document:

1. **Event Trigger**: What GitHub event triggers the workflow (push, pull_request, issues, schedule, workflow_dispatch)
2. **Workflow File**: Link to the voedger workflow file
3. **Conditions**: Any conditions that must be met
4. **Steps**: Numbered list of workflow steps with:
   - Link to workflow file with line range
   - Description of what the step does
   - Parameters passed to called workflows
   - Links to detailed sections for complex workflows

### Special Requirements

- **Script Details**: When a workflow calls a bash script (e.g., `add-issue-commit.sh`, `close-issue.sh`, `linkmilestone.sh`), replace generic descriptions with detailed substeps showing:
  - Input validation
  - Processing logic
  - Output/actions taken
  - Make script names clickable links to the actual script files in ci-action repository

- **Reusable Workflows**: When multiple events call the same reusable workflow (e.g., `ci_reuse_go.yml`):
  - Create a dedicated "<External repo name> Reusable Workflows" section with full details
  - Link to this section from event sections instead of duplicating content
  - Use anchor links like `→ See [workflow-name details](#anchor-link)`

- **Service Dependencies**: When workflows use services (e.g., ScyllaDB, DynamoDB Local):
  - Make service descriptions clickable links to the service configuration in the workflow file

- **Links Format**:
  - Current repository files: relative paths (e.g., `ci-pkg-cmd.yml#L3-L9`)
  - External repository files: full GitHub links (e.g., `https://github.com/untillpro/ci-action/blob/main/.github/workflows/cp.yml#L36-L40`)
  - Internal sections: anchor links (e.g., `#domergeprsh---auto-merge-script`)

### Organization

- Group related events together
- Use horizontal rules (`---`) to separate sections
- Use subsections (`###`) for detailed workflow steps
- Use bold (`**text**`) only for subsection headers, not for step names

---

## EXECUTION_FLOW_DIAGRAMS.md Requirements

### Diagram Types

1. **Overview Diagram** (Flowchart)
   - Shows all GitHub events
   - Shows how events trigger workflows
   - Shows workflow dependencies and calls
   - Shows final data outputs
   - Use color coding for different categories

2. **Sequence Diagrams** (One per major workflow)
   - Show step-by-step execution flow
   - Show data flow between components
   - Show conditional logic (alt/else branches)
   - Show success/failure paths

### Diagram Content

- **Participants**: GitHub events, workflows, scripts, services, outputs
- **Interactions**: Show calls, returns, and data flow
- **Details**: Include parameters, environment variables, and key actions
- **Accuracy**: Reflect exact steps from TRACE.md

### Mermaid Format

- Use `graph TD` for flowcharts
- Use `sequenceDiagram` for sequence diagrams
- Include styling with colors for visual clarity
- Add descriptive titles

---

## Key Principles

1. **Accuracy**: All information must match the actual workflow files
2. **Completeness**: Document all workflows and events
3. **Clarity**: Use clear, concise descriptions
4. **DRY**: Avoid duplicating information; use links instead
5. **Traceability**: Every step should be linkable to source code
6. **Consistency**: Use consistent formatting and terminology
7. **Maintainability**: Structure should make updates easy

---

## Workflow Analysis Checklist

- [ ] Identify all GitHub events that trigger workflows
- [ ] Map each event to its corresponding workflow file
- [ ] Document all workflow steps and their purposes
- [ ] Identify all workflow calls (uses, workflow_call)
- [ ] Document all bash scripts and their actions
- [ ] Identify service dependencies
- [ ] Map data flow between components
- [ ] Identify conditional logic and branches
- [ ] Document failure handling
- [ ] Create visual diagrams for complex flows
- [ ] Verify all links point to correct line ranges
- [ ] Check for duplicate content and consolidate

---

## Output Files

1. **TRACE.md**: Comprehensive text documentation with links
2. **EXECUTION_FLOW_DIAGRAMS.md**: Visual diagrams in Mermaid format

Both files should be placed in `.github/workflows/` directory.

