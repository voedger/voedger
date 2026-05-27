Feature: Authentication

  Client establishes an authenticated identity for a user or device and receives
  principal tokens that can be presented to Voedger APIs.

  Rule: Login creation

    Scenario: Client creates a user login from a verified email token
      Given Client has a valid verified email token
      When Client creates a user login with display name and password
      Then the response status is "201 Created"
      And the user login is accepted
      And the user profile workspace creation is started

    Scenario: Client creates a device login
      When Client creates a device login for an application
      Then the response status is "201 Created"
      And the response contains generated device login and password
      And the device profile workspace creation is started

    Scenario: Login creation rejects duplicate login
      Given a login already exists
      When Client creates the same login again
      Then the response status is "409 Conflict"

    Scenario Outline: User login creation rejects malformed request
      When Client creates a user login without "<field>"
      Then the response status is "400 Bad Request"
      And the response indicates "<field>" is missing

      Examples:
        | field              |
        | verifiedEmailToken |
        | displayName        |
        | password           |

    Scenario: Device login creation rejects request body
      When Client creates a device login with a request body
      Then the response status is "400 Bad Request"
      And the response indicates unexpected body

  Rule: Sign-in and profile readiness

    Scenario Outline: Subject signs in after profile workspace is ready
      Given "<subject>" login exists
      And the profile workspace for "<subject>" is ready
      When Client signs in with login and password
      Then the response contains principalToken, expiresInSeconds, and profileWSID

      Examples:
        | subject |
        | user    |
        | device  |

    Scenario: Sign-in reports profile workspace not ready
      Given a login exists
      And the profile workspace for the login is not ready
      When Client signs in with login and password
      Then the response status is "409 Conflict"
      And the response indicates the profile workspace is not yet ready

    Scenario: Sign-in reports profile workspace creation error
      Given a login exists
      And profile workspace creation failed for the login
      When Client signs in with login and password
      Then the response indicates the profile workspace creation error

  Rule: Principal token contract

    Scenario Outline: Principal token carries authn identity fields
      Given "<subject>" login exists
      And the profile workspace for "<subject>" is ready
      When Client signs in with login and password
      Then the issued principal token identifies login, subject kind, and profileWSID

      Examples:
        | subject |
        | user    |
        | device  |

    Scenario: Principal token uses default TTL when no custom TTL is requested
      Given a login exists
      When Client signs in with login and password
      Then expiresInSeconds matches the default principal token expiration

    Scenario: Principal token rejects TTL above the maximum
      Given a login exists
      When Client requests a principal token with TTL above the maximum
      Then the response status is "400 Bad Request"
      And the response indicates the maximum token TTL

    Scenario: Client refreshes a principal token
      Given Client has a valid principal token
      When Client refreshes the principal token
      Then the response contains a new principalToken
      And the new principalToken preserves the authn identity fields

  Rule: Password lifecycle

    Scenario: Client changes user password
      Given a user login exists
      When Client changes the password with the current password
      Then the response status is "200 OK"
      And Client can sign in with the new password

    Scenario: Password change rejects malformed request
      When Client changes a password without login, oldPassword, or newPassword
      Then the response status is "400 Bad Request"

    Scenario: Password change rejects unknown login or wrong current password
      When Client changes a password for an unknown login or with the wrong current password
      Then the response status is "401 Unauthorized"

    Scenario: Client resets password by verified email
      Given a user login exists
      When Client initiates password reset by email
      And Client verifies the reset code
      And Client resets the password with the verified value token
      Then Client can sign in with the new password

    Scenario: Password reset initiation rejects unknown login
      When Client initiates password reset for an unknown login
      Then the response status is "400 Bad Request"

    Scenario: Password reset verification rejects wrong verification code
      Given Client initiated password reset by email
      When Client verifies the reset code with a wrong code
      Then the response status is "400 Bad Request"

  Rule: Exception flows

    Scenario: User login creation rejects an invalid verified email token
      Given Client has an invalid verified email token
      When Client creates a user login with display name and password
      Then the response status is "400 Bad Request"
      And the response indicates verifiedEmailToken validation failed

    Scenario: Login creation rejects an invalid login name
      When Client creates a login with an invalid login name
      Then the response status is "400 Bad Request"
      And the response indicates incorrect login format

    Scenario: Sign-in rejects unknown login or wrong password
      When Client signs in with unknown login or wrong password
      Then the response status is "401 Unauthorized"

    Scenario: Principal token refresh requires an existing token
      Given Client has no principal token
      When Client refreshes the principal token
      Then the response status is "401 Unauthorized"
