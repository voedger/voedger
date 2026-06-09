Feature: Pull request reviews

  NonWriters and Writers receive automated feedback on pull requests before merge.

  Rule: Automatic reviews

    Scenario Outline: Pull request lifecycle event creates an automated review
      Given Pull Request "AIR-4201" is "<state>"
      When GitHub reports that the pull request is "<event>"
      Then Pull Request "AIR-4201" has an automated review
      And the review uses repository review rules

      Examples:
        | state     | event             |
        | draft     | opened            |
        | non-draft | updated           |
        | non-draft | reopened          |
        | draft     | ready for review  |

  Rule: Manual review requests

    Scenario: Writer requests another review with extra instructions
      Given Pull Request "AIR-4201" exists
      When Writer comments "/review after latest changes" on Pull Request "AIR-4201"
      Then Pull Request "AIR-4201" has a new automated review
      And the review uses repository review rules
      And the review uses "after latest changes" as extra review instructions

    Scenario: Writer requests another review without extra instructions
      Given Pull Request "AIR-4201" exists
      When Writer comments "/review" on Pull Request "AIR-4201"
      Then Pull Request "AIR-4201" has a new automated review
      And the review has no extra review instructions

    Scenario: NonWriter requests another review
      Given Pull Request "AIR-4201" exists
      When NonWriter comments "/review" on Pull Request "AIR-4201"
      Then Pull Request "AIR-4201" has no new automated review
      And NonWriter is told that only Writer can request an automated review

  Rule: Ignored comments

    Scenario Outline: Comment does not trigger review
      Given Pull Request "AIR-4201" exists
      When Writer creates a pull request comment with body "<comment>"
      Then Pull Request "AIR-4201" has no new automated review

      Examples:
        | comment                    |
        | Please run /review         |
        | I reviewed this manually   |
        | /reviewed by reviewer      |

    Scenario Outline: Existing comment change does not trigger review
      Given Pull Request "AIR-4201" exists
      When Writer "<action>" an existing pull request comment with body "/review"
      Then Pull Request "AIR-4201" has no new automated review

      Examples:
        | action  |
        | edits   |
        | deletes |

    Scenario: Review command on an issue is ignored
      Given Issue "AIR-4202" is not a pull request
      When Writer comments "/review" on Issue "AIR-4202"
      Then Issue "AIR-4202" has no automated review
