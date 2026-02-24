# Templates: Implementation Plan

## Implementation Plan template

Level-1 header format:

```markdown
# Implementation plan: {Change request title}
```

Section order (each section is optional; when present, must appear in this order):

- Functional design
- Provisioning and configuration
- Technical design
- Construction
- Quick start

Note: Sections (Functional design, Technical design, Construction) contain checkbox lists that reference files to create or update. Do not put the actual design content there - actual content goes into separate files.

### Section: Functional design

Ref. `{templates_folder}/tmpl-fd.md` for specification file formats.

### Section: Technical design

Ref. `{templates_folder}/tmpl-td.md` for specification file formats.

### Section: Provisioning and configuration

Rules:

- Always prefer to use CLI commands
- For provisioning
  - Make sure that required components are not already installed
  - Specify latest possible stable version, always use web search to find it
  - Detect current OS - provide OS-specific instructions only
  - Group by category
  - Prefer vendor-independent alternatives when available

Example:

```markdown
**Provisioning:**

- [ ] install: Docker 24.0+
  - `winget install Docker.DockerDesktop` or `https://docs.docker.com/get-docker/`

**Configuration:**

- [ ] update: [package.json](../../package.json): Add express web framework
  - `npm install express`

- [ ] update: [tsconfig.json](../../tsconfig.json): Enable strict mode
  - `Manual edit - Set strict: true`
```

### Sections: Functional design, Technical design, Construction

These sections contain checkbox lists referencing files to create or update.

Format:

```markdown
- [ ] {action}: [{folder}/{filename}](relative-path)
  - {action}: Description of changes
```

Rules:

- Always use actual relative paths from the Change File to particular file (e.g., ../../specs/domain/myctx/my.feature)
- Use relative paths for both existing files and new files that don't exist yet
- Technical design section
  - Reference existing architecture files (e.g., `../../specs/prod/apps/vvm-orch--arch.md`) when updating them
  - Use templates from `{templates_folder}/tmpl-td.md` for structure of new files
- Construction section
  - If design sections exist, run `git diff <baseline> -- {specs_folder}/` to identify exact spec changes (baseline from Change File frontmatter)
  - List all non-specification files that need to be created or modified, not already covered by other sections
  - Includes source files, tests, documentation, scripts, configuration - any file changes
  - Optional grouping: when items span 3+ distinct dependency categories, group under `###` headers ordered by dependency (foundational changes first, dependent changes after)

Example:

```markdown
## Functional design

- [ ] update: [myctx/my.feature](../../specs/domain/myctx/my.feature)
  - add: Branch push validation (main blocked, feature allowed)

## Technical design

- [ ] update: [apps/vvm-orch--arch.md](../../specs/prod/apps/vvm-orch--arch.md)
  - update: Leadership renewal interval documentation (1s instead of TTL/2)

## Construction

- [ ] update: [internal/auth/validator.go](../../../internal/auth/validator.go)
  - fix: null pointer when validating empty email field
- [ ] update: [internal/auth/validator_test.go](../../../internal/auth/validator_test.go)
  - add: Test case for empty email validation
- [ ] update: [README.md](../../../README.md)
  - update: supported Go version to 1.21+
- [ ] create: [scripts/migrate-db.sh](../../../scripts/migrate-db.sh)
  - add: Database migration script for auth schema changes
```

Example with optional grouping:

```markdown
## Construction

### Schema changes

- [ ] update: [internal/db/migrations/003_add_roles.sql](../../../internal/db/migrations/003_add_roles.sql)
  - add: roles table and user_roles junction table

### Function signature changes

- [ ] update: [internal/auth/service.go](../../../internal/auth/service.go)
  - update: AuthenticateUser to accept RoleChecker parameter
- [ ] update: [internal/auth/middleware.go](../../../internal/auth/middleware.go)
  - update: WithAuth middleware to use new AuthenticateUser signature

### Caller updates

- [ ] update: [cmd/server/main.go](../../../cmd/server/main.go)
  - update: wire RoleChecker into AuthenticateUser calls

### Tests

- [ ] update: [internal/auth/service_test.go](../../../internal/auth/service_test.go)
  - add: test cases for role-based authentication
```

### Section: Quick start

- When to include: New features, APIs, CLI commands, or configuration changes that users need to learn how to use
- Skip if: Internal refactoring, bug fixes, or changes with no user-facing impact
- Show how to use the feature with minimal examples

#### Example

Run migration with dry-run mode:

```bash
./scripts/migrate-db.sh --dry-run up
```

Generate report with date filter:

```bash
npm run report -- --from=2024-01-01 --to=2024-12-31
```
