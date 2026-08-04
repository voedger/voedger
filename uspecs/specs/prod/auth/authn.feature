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

    Scenario: Login creation rejects an active duplicate login
      Given an active login already exists
      When Client creates the same login again
      Then the response status is "409 Conflict"

    Scenario: Login creation succeeds for a deactivated login name
      Given a login was previously created and is now deactivated
      When Client creates a login with the same name again
      Then the response status is "201 Created"
      And a new login is accepted with a fresh profile workspace
      And the previously deactivated login is no longer reachable for sign-in or token issue

    Scenario: Login creation rejects an existing active alias
      Given a login exists with an active login alias
      When Client creates a login using that alias value
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

  Rule: Login alias management

    Scenario: System sets the first Login Alias
      Given a User Login "jsmith" with no Login Alias
      When System sets the Login Alias "j.smith" for "jsmith"
      Then "jsmith" has the active Login Alias "j.smith"

    Scenario: System replaces an existing Login Alias
      Given a User Login "jsmith" with the active Login Alias "j.smith"
      When System sets the Login Alias "john.smith" for "jsmith"
      Then "jsmith" has the active Login Alias "john.smith"
      And "j.smith" is no longer active

    Scenario: System clears a Login Alias
      Given a User Login "jsmith" with the active Login Alias "j.smith"
      When System clears the Login Alias for "jsmith"
      Then "jsmith" has no active Login Alias

    Scenario: Alias management rejects caller without System Principal Token
      Given a user login exists
      When a caller without a System Principal Token sets a login alias for the user
      Then the alias change is rejected

    Scenario Outline: Alias creation or update rejects a colliding identifier
      Given a user login exists
      And another "<identifier>" exists in the same application
      When System "<operation>" the user's login alias using that value
      Then the alias change is rejected as a conflict

      Examples:
        | operation | identifier   |
        | creates   | login        |
        | creates   | active alias |
        | updates   | login        |
        | updates   | active alias |

    Scenario: Alias creation rejects an invalid sign-in identifier
      Given a user login exists
      When System sets an invalid login alias for the user
      Then the alias change is rejected
      And the response indicates incorrect login format

  Rule: Login state visibility

    Scenario Outline: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner
      Given User Login "jsmith" exists
      When <caller> reads the Login CDoc of User Login "jsmith"
      Then the read <result>

      Examples:
        | caller                                             | result      |
        | System                                             | succeeds    |
        | a WorkspaceOwner of the target registry workspace | succeeds    |
        | a WorkspaceOwner of another workspace             | is rejected |
        | a caller with neither authorization                | is rejected |

    Scenario: A target registry WorkspaceOwner read returns Login state
      Given User Login "jsmith" has active LoginAlias "j.smith", no alias change in progress, and no alias error
      And CanonicalLoginEnablement of User Login "jsmith" is Disabled
      When a WorkspaceOwner of the target registry workspace reads the Login CDoc of User Login "jsmith"
      Then the Login CDoc indicates CanonicalLoginEnablement is Disabled
      And Alias is "j.smith"
      And AliasInProc is 0
      And AliasError is empty

  Rule: Canonical Login enablement management

    Scenario Outline: System sets CanonicalLoginEnablement idempotently
      Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
      When System <operation> the canonical Login "jsmith" twice
      Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"

      Examples:
        | initial state | operation | resulting state |
        | Enabled       | disables  | Disabled        |
        | Disabled      | disables  | Disabled        |
        | Disabled      | enables   | Enabled         |
        | Enabled       | enables   | Enabled         |

    Scenario Outline: Canonical Login enablement management requires a System PrincipalToken
      Given CanonicalLoginEnablement of User Login "jsmith" is Enabled
      When a caller without a System PrincipalToken <operation> the canonical Login "jsmith"
      Then the enablement operation is rejected
      And CanonicalLoginEnablement of User Login "jsmith" remains Enabled

      Examples:
        | operation |
        | disables  |
        | enables   |

  Rule: Disabled canonical Login behavior

    Scenario: Disabling canonical Login preserves its active LoginAlias
      Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
      When System disables the canonical Login "jsmith@example.com"
      Then CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
      And LoginAlias "j.smith@example.com" remains active for User Login "jsmith@example.com"

    Scenario Outline: Disabled canonical Login rejects only canonical entry operations
      Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
      When Client <operation>
      Then the response status is "<status>"
      And the response is the same as for <public failure>
      And <observable result>

      Examples:
        | operation                                                                      | status           | public failure                     | observable result                              |
        | signs in using canonical Login "jsmith@example.com" and the correct password   | 401 Unauthorized | an unknown Login or wrong password | no PrincipalToken is returned                  |
        | initiates password reset using canonical Login "jsmith@example.com"            | 400 Bad Request  | an unknown Login                   | no password-reset verification code is issued  |

    Scenario: Active LoginAlias sign-in is unaffected by canonical Login disablement
      Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
      And CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
      And ProfileWorkspace of User Login "jsmith@example.com" is ready
      When Client signs in using LoginAlias "j.smith@example.com" and the correct password
      Then the response contains PrincipalToken, expiresInSeconds, and profileWSID

    Scenario: Active LoginAlias password reset is unaffected by canonical Login disablement
      Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
      And CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
      When Client initiates password reset using LoginAlias "j.smith@example.com"
      And Client verifies the reset code sent to "j.smith@example.com"
      And Client resets the password with the VerifiedValueToken
      Then Client can sign in using LoginAlias "j.smith@example.com" and the new password
      And CanonicalLoginEnablement of User Login "jsmith@example.com" remains Disabled

    Scenario: Password reset initiated before canonical Login disablement can complete
      Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
      And Client initiated password reset using canonical Login "jsmith@example.com"
      And Client verified the reset code and received a VerifiedValueToken
      And System disabled the canonical Login "jsmith@example.com"
      When Client resets the password with the VerifiedValueToken
      Then Client can sign in using active LoginAlias "j.smith@example.com" and the new password
      And CanonicalLoginEnablement of User Login "jsmith@example.com" remains Disabled

    Scenario: Disabled canonical identifier remains reserved
      Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
      When Client creates User Login "jsmith@example.com" again
      Then the response status is "409 Conflict"

    Scenario Outline: Re-enabling canonical Login restores canonical entry operations
      Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
      When System enables the canonical Login "jsmith@example.com"
      And Client <operation>
      Then <observable result>

      Examples:
        | operation                                                                     | observable result                            |
        | signs in using canonical Login "jsmith@example.com" and the existing password | a new PrincipalToken is returned             |
        | initiates password reset using canonical Login "jsmith@example.com"           | a password-reset verification code is issued |

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

    Scenario: User signs in with original login while alias is active
      Given a user login exists with an active login alias
      And the profile workspace for the user is ready
      When Client signs in with original login and password
      Then the response contains principalToken, expiresInSeconds, and profileWSID

    Scenario: User signs in with active alias
      Given a user login exists with an active login alias
      And the profile workspace for the user is ready
      When Client signs in with alias and password
      Then the response contains principalToken, expiresInSeconds, and profileWSID

    Scenario: Sign-in rejects a previous alias after alias update
      Given a user login exists
      And the profile workspace for the user is ready
      And System updated the user's login alias
      When Client signs in with the previous alias and password
      Then the response status is "401 Unauthorized"

    Scenario: Sign-in rejects a cleared alias
      Given a user login exists
      And the profile workspace for the user is ready
      And System cleared the user's login alias
      When Client signs in with the cleared alias and password
      Then the response status is "401 Unauthorized"

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
      Then the issued principal token identifies its login (the canonical login), subject kind, and profileWSID

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
      And the new principalToken preserves the login (canonical), alias, subject kind, and profileWSID from the input token

    Scenario Outline: Principal token carries the canonical login and the active alias
      Given a user login that <alias state>
      And the profile workspace for the user is ready
      When Client signs in with <identifier> and password
      Then the issued principal token's login is the canonical login
      And its alias is <alias value>

      Examples:
        | alias state         | identifier     | alias value      |
        | has an active alias | alias          | the active alias |
        | has an active alias | original login | the active alias |
        | has no active alias | original login | empty            |

    Scenario: Existing principal token retains login and alias after alias changes
      Given Client has a valid principal token issued while a login alias is active
      When System updates or clears that login alias
      Then the existing principal token remains valid until normal expiration
      And the existing principal token retains the login (canonical) and alias captured at issue time

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

    Scenario: Client resets password by verified alias email
      Given User Login "jsmith" has active Login Alias "j.smith@example.com"
      When Client initiates password reset using Login Alias "j.smith@example.com"
      And Client verifies the reset code sent to "j.smith@example.com"
      And Client resets the password with the verified value token
      Then Client can sign in as User Login "jsmith" with the new password

    Scenario Outline: Password reset initiation rejects an inactive alias
      Given User Login "jsmith" had Login Alias "j.smith@example.com"
      And System <operation> that Login Alias
      When Client initiates password reset using Login Alias "j.smith@example.com"
      Then the response status is "400 Bad Request"

      Examples:
        | operation             |
        | replaced              |
        | cleared               |

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

    Scenario: Sign-in rejects a deactivated login with the same error as a missing login
      Given a login exists but is deactivated
      When Client signs in with that login and password
      Then the response status is "401 Unauthorized"
      And the response indicates the login or password is incorrect

    Scenario: Principal token refresh requires an existing token
      Given Client has no principal token
      When Client refreshes the principal token
      Then the response status is "401 Unauthorized"
