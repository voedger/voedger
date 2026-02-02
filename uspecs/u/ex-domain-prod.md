<!-- Example -->

# Domain: AI-assisted software engineering

## System

Scope:

- Tools and workflows to assist software engineers in designing, specifying, and constructing software systems using AI agents
- Supports both greenfield and brownfield projects

Key features:

- Quick design: no per-project installation or configuration required, great for prototyping and experimentation
- Greenfield and brownfield projects support
- Optional simplified workflow for brownfield projects
- Gherkin language for functional specifications
- Maintaining actual functional specifications
- Maintaining actual architecture and technical design
- Working with multiple domains: by default `prod` and `devops`, can be extended with custom domains

## External actors

Roles:

- 👤Engineer
  - Software engineer interacting with the system

Systems:  

- ⚙️AI Agent
  - System that can follow text based instructions to complete multi-step tasks

## Concepts

- Change Request: a formal proposal to modify System
- Active Change Request: a Change Request that is being actively worked on
- Functional Design
  - A functional specification focuses on what various outside agents (people using the program, computer peripherals, or other computers, for example) might "observe" when interacting with the system ([stanford](https://web.archive.org/web/20171212191241/https://uit.stanford.edu/pmo/functional-design))
- Technical Design
  - The functional design specifies how a program will behave to outside agents and the technical design describes how that functionality is to be implemented ([stanford](https://web.archive.org/web/20241111203113/https://uit.stanford.edu/pmo/technical-design))
- Construction
  - Software construction refers to the detailed creation and maintenance of software through coding, verification, unit testing, integration testing and debugging (SWEBOK, 2025, chapter 4)

## Contexts

### conf

Install and maintain the System.

Relationships with external actors:

- 🎯conf ->|installation| 👤Engineer
- 🎯conf -> |configuration| ⚙️AI Agent
  - AI Agent parameters configuration

### softeng

Software engineering through human-AI collaborative workflows.

Relationships with external actors:

- 🎯softeng -> 👤Engineer
  - Change request management
  - Functional design assistance
  - Architecture and technical design assistance
  - Construction assistance

## Context map

- conf -> |parameters| softeng
