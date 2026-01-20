# Templates

---

## change.md

Example:

```markdown
# Change request: {Change request title}

- issue: [ISSUE-ID: Issue name](link)  (if specified)

## Problem

1-3 sentences describing the problem. Focus on the need, avoid implementation details.

## Solution overview

What is being proposed to solve the problem. Start with introductory sentence, follow with capabilities list if applicable.
```

Rules:

- YAML front matter with uspecs.* prefixed keys is added automatically by uspecs.sh script
- If issue is not specified skip the line `issue: [ISSUE-ID: Issue name](link)`
- If issue is specified, try to fetch issue content to fill in Issue attribute, Problem and Solution overview sections

---

## Functional Design Specifications

### Scenarios File (.feature) template

Rules:

- Prefer Scenario Outlines with Examples tables over multiple similar Scenarios
- Use data tables in steps for inline structured data

```gherkin
Feature: Payment processing
  Waiter process customer payments using methods configured by Location Owner (LO)

  Scenario: LO creates payment method
    When LO creates payment method "Cash" with valid fields per req.md#create-payment-fields
    Then payment method "Cash" should be available for waiters

  Scenario: LO disables payment method
    When LO deactivates payment method "Credit Card"
    Then "Credit Card" should not be available for waiters

  Scenario Outline: Waiter processes payment
    Given payment method "<method>" with availability "<availability>" and max_amount "<max_amount>"
    When Waiter processes payment of "<amount>" with "<method>"
    Then payment should <result>

    Examples:
      | method      | availability | max_amount | amount | result                                   |
      | Cash        | true         | 50.00      | 49.00  | succeed                                  |
      | Cash        | true         | 50.00      | 75.00  | be rejected with "Exceeds maximum limit" |
      | Credit Card | false        | 100.00     | 50.00  | be rejected with "Method not available"  |
```

### Requirements File (*reqs.md) template

Contain requirements that do not fit into Gherkin scenarios.

```markdown
# Requirements: Payment processing

## Create payment fields

| field       | type    | required |
|-------------|---------|----------|
| name        | string  | yes      |
| fee_percent | decimal | no       |
| max_amount  | decimal | no       |

Validation rules:

- Unknown fields: Any field not listed above should be rejected
- max_amount: Must be positive if specified
```
