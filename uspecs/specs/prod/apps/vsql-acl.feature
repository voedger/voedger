Feature: VSQL SELECT ACL on projected and WHERE fields

  ACL is enforced on every field referenced by a VSQL SELECT statement,
  including fields used inside the WHERE clause. A field for which SELECT
  is not allowed must not be readable directly nor implicitly via a filter.

  Background:
    Given table "air.Orders" with fields "Id", "Total", "CustomerId"
    And role "Reader"
    And grant "GRANT SELECT(Id, Total) ON TABLE air.Orders TO Reader"
    And revoke "REVOKE SELECT(CustomerId) ON TABLE air.Orders FROM Reader"
    And VADeveloper is authenticated with role "Reader"

  Rule: SELECT is denied when any referenced field has SELECT denied

    Scenario: Denied field in WHERE with allowed projection
      When VADeveloper executes "select Id from air.Orders where CustomerId = 123"
      Then response status is "403 Forbidden"

    Scenario: Denied field in WHERE combined with allowed field via AND
      When VADeveloper executes "select Id from air.Orders where Total > 0 and CustomerId = 123"
      Then response status is "403 Forbidden"

    Scenario: Denied field in WHERE combined with allowed field via OR
      When VADeveloper executes "select Id from air.Orders where Total > 0 or CustomerId = 123"
      Then response status is "403 Forbidden"

    Scenario: Denied field in nested WHERE expression
      When VADeveloper executes "select Id from air.Orders where (Total > 0 and (CustomerId = 123 or Id = 1))"
      Then response status is "403 Forbidden"

    Scenario: Denied field in projection
      When VADeveloper executes "select CustomerId from air.Orders where Id = 1"
      Then response status is "403 Forbidden"

    Scenario: Star projection with any denied field on the table
      When VADeveloper executes "select * from air.Orders where Id = 1"
      Then response status is "403 Forbidden"

  Rule: SELECT is allowed when every referenced field has SELECT allowed

    Scenario: Allowed projection and allowed WHERE
      When VADeveloper executes "select Id, Total from air.Orders where Total > 0 and Id = 1"
      Then matching records are returned

    Scenario: Allowed projection without WHERE
      When VADeveloper executes "select Id, Total from air.Orders"
      Then matching records are returned

  Rule: ACL on WHERE fields applies to views

    Scenario: Denied view key field in WHERE
      Given view "air.OrdersByCustomer" with key fields "Year", "CustomerId"
      And grant "GRANT SELECT(Year) ON VIEW air.OrdersByCustomer TO Reader"
      And revoke "REVOKE SELECT(CustomerId) ON VIEW air.OrdersByCustomer FROM Reader"
      When VADeveloper executes "select Year from air.OrdersByCustomer where Year = 2026 and CustomerId = 123"
      Then response status is "403 Forbidden"
