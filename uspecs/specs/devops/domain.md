# Domain: Voedger engineering operations

Development, review, delivery, and maintenance workflows used to change and operate the Voedger repository.

## Overview

Scope:

- NonWriter and Writer workflows around repository changes
- Automated checks and review support that run before changes are merged
- Repository automation that improves engineering feedback without changing the Voedger production runtime

Key features:

- Development workflow automation
- CI and validation workflows
- Engineering rules and guidance for human developers and AI agents

## External actors

Roles:

- 👤NonWriter
  - User below Write repository permission who can open and update pull requests
- 👤Writer
  - User with Write or higher repository permission who reviews pull requests and decides whether changes can be merged

Systems:

- ⚙️GitHub
  - Hosts the repository, pull requests, comments, secrets, and workflow execution

---

## Concepts

- `Pull Request`
  - A proposed repository change submitted for automated and human review before merge

- `Review`
  - Feedback attached to a pull request, including summary findings and inline comments

- `Repository Rule`
  - Versioned development guidance stored in the repository for contributors, automation, and agents

- `Repository Secret`
  - Secret value managed by GitHub Actions and used by trusted workflows without committing credentials to the repository

---

## Contexts

- **dev**
  - Development workflow automation, including CI validation and repository guidance for contributors

### Context map

```mermaid
graph
  dev["📦dev"]
```

### dev

Relationships:

```mermaid
graph
  dev["📦dev"]
  NonWriter["👤NonWriter"]
  Writer["👤Writer"]
  GitHub["⚙️GitHub"]

  NonWriter -->|opens and updates pull requests| dev
  Writer -->|reviews pull requests| dev
  GitHub -->|emits workflow events| dev
```
