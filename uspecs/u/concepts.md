# Concepts

## Self-evident

- Computer System (System), Operation, Rule, Concept, Model

## Changes

- Change Request: a formal proposal to modify System
- Active Change Request: a Change Request that is being actively worked on

## Misc

- External Actor: interacts with the Computer System
  - Role (human user)
  - System (external system/integration)

## DDD

- Domain: targeted subject area of a computer system
  - Pre-existing domains are `prod` and `devops`
    - `prod`: The business logic and customer-facing capabilities of the product - what the product does for its users
    - `devops`: development, testing, delivery, deployment, maintenance (monitoring, observability etc.) aspects of the product
- Object (Domain Object): one of
  - Entity: has identity and lifecycle (e.g., User, Order, Article)
  - Value Object: defined by attributes, no identity (e.g., Address)
  - Service: encapsulates operations over multiple objects  
- Context: a specific area within a domain with specific set of actors, concepts, operations and rules
  - Primary indicators
    - Low coupling to other contexts
    - Autonomy of evolution (components evolve independently)
    - Team/organizational responsibility
    - Data autonomy
  - Naming: noun (normally plural) or noun phrase
    - Examples: `payments`, `menu`
- Feature: Cohesive set of scenarios within a context
  - Single object: operations on the same object
  - Cross-object: related operations across multiple objects (workflow)
  - Can involve multiple actors
  - Naming:
    - Single object: noun-op, e.g., `category-mgmt`, `account-mgmt`
    - Cross-object: op-noun, e.g., `import-menu`, `transfer-funds`
  - Context contains Features, Feature belongs to exactly one Context
  - Context defines WHAT (entities/nouns), Feature defines HOW (actions/verbs)
