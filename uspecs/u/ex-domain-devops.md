<!-- Example -->

# Domain: devops

## System

Tools, scripts and configuration files to assist with development, testing, deployment, operation.

## External actors

Roles:

- 👤Developer
  - Can modify codebase
- 👤Maintainer
  - Can make releases

Systems:  

- ⚙️GitHub
  - A platform that allows to store, manage, share code and automate related workflows

## Context map

- 🎯dev -> |supplier-customer| 🎯ops
  - Deployment automation and tooling

## Contexts

### dev

Development, testing, and release automation.

Relationships with external actors:

- 🎯dev -> |supplier-customer| 👤Developer
  - Development tooling and workflows
  - Test tooling and workflows
- 🎯dev -> |supplier-customer| 👤Maintainer
  - Release management tooling and workflows
- ⚙️GitHub -> |supplier-customer| 🎯dev
  - Repository hosting
  - CI/CD automation

### ops

Production operations, monitoring, and incident response.
