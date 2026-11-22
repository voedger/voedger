Feature: Async projector entire log run logging

  Background:
    Given projector test.LoggingProjector belongs to test1/app1 partition 1

  Scenario: A new projector logs entire log run start and completion
    Given a new projector has events through offset 3 and verbose logging is enabled
    When the projector finishes the entire log run from FirstOffset
    Then one INFO ap.entire_log_run.start entry is present
    And one INFO ap.entire_log_run.complete entry has nonnegative duration_ms
    And both entries identify the application, projector and partition without message text or args
    And neither entry includes poffset
    And entire log run event details are absent

  Scenario: Entire log run completion waits for pending processing
    Given a new projector has read through offset 3 but its last event is still processing
    When the operator inspects the logs
    Then ap.entire_log_run.start is present and ap.entire_log_run.complete is absent

  Scenario: An entire log run retry keeps one start and one completion
    Given a new projector fails its first attempt to process offset 2
    When the entire log run succeeds after an automatic retry
    Then one INFO ap.entire_log_run.start precedes the error and one INFO ap.entire_log_run.complete follows it
    And the ERROR ap.error entry identifies the application, projector, workspace, event and offsets without args
