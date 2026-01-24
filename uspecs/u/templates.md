# Templates

---

## Change File Template 1

Format:

```markdown
# Change request: {Change request title}

## Why

{1-3 sentences describing the reason, cause, purpose, or belief behind the change request}

## What

{What is being delivered} 
```

In What section organize content with introductory sentences followed by bullet points:

```markdown
Introductory sentence1:

- Item 1
- Item 2

Introductory sentence2:

- Item 1
- Item 2
```

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
