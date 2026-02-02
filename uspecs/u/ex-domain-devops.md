<!-- Example -->

# Domain: Development and operations

## System

Tools, scripts, and configuration files to assist with development, testing, deployment, and operations

## External actors

Roles:

- 👤Developer
  - Can modify codebase
- 👤Maintainer
  - Can make releases

Systems:  

- ⚙️GitHub
  - A platform that allows to store, manage, share code and automate related workflows

---

## Contexts

### dev

Development, testing, and release automation.

Relationships with external actors:

- 🎯dev -> 👤Developer
  - Development tooling and workflows
  - Test tooling and workflows
- 🎯dev -> 👤Maintainer
  - Release management tooling and workflows
- ⚙️GitHub -> 🎯dev
  - Repository hosting
  - CI/CD automation

### ops

Production operations, monitoring, and incident response.

---

## Context map

- dev -> ops
  - Deployment automation and tooling
