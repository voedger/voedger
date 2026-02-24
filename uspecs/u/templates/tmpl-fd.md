# Templates: Functional Design

## Scenarios File (.feature) template

Rules:

- Create very concise scenarios
- Focus on user-facing behavior (what the user observes), not internal implementation steps
- Place validation errors, failures, and error recovery under `Rule: Edge cases`
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

  Rule: Edge cases

    Scenario: Waiter processes payment with no payment methods configured
      Given no payment methods are configured for the location
      When Waiter attempts to process a payment
      Then error "No payment methods available" is displayed
      And payment is not created
```

## Requirements File (*reqs.md) template

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
