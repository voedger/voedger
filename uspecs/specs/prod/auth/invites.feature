Feature: Workspace invitations

  Workspace owners invite users with application roles, and invitees manage membership under their canonical login.

  Rule: Sending invitations

    Scenario: Workspace owner sends an invitation
      Given Workspace "Acme" exists
      And User Login "alice@example.com" exists
      When Workspace Owner invites "alice@example.com" to Workspace "Acme" with Role "app1pkg.LimitedAccessRole"
      Then "alice@example.com" receives an invitation email for Workspace "Acme"
      And Workspace "Acme" has a pending invitation for "alice@example.com" with Role "app1pkg.LimitedAccessRole"

    Scenario: Workspace owner resends a pending invitation
      Given Workspace "Acme" has a pending invitation for "alice@example.com"
      When Workspace Owner resends the invitation
      Then "alice@example.com" receives a new invitation verification code
      And the pending invitation remains for Workspace "Acme"

    Scenario: Workspace owner changes roles while resending a pending invitation
      Given Workspace "Acme" has a pending invitation for "alice@example.com" with Role "app1pkg.LimitedAccessRole"
      When Workspace Owner resends the invitation with Role "app1pkg.SpecialAPITokenRole"
      Then the pending invitation has Role "app1pkg.SpecialAPITokenRole"

    Scenario: Workspace owner cancels a pending invitation
      Given Workspace "Acme" has a pending invitation for "alice@example.com"
      When Workspace Owner cancels the invitation
      Then the response status is "400 Bad Request" when User Login "alice@example.com" tries to accept it
      And User Login "alice@example.com" is not a member of Workspace "Acme"

    Scenario: Workspace owner cannot cancel a non-existing invitation
      Given Workspace "Acme" has no invitation with ID "66048"
      When Workspace Owner cancels invitation with ID "66048"
      Then the response status is "400 Bad Request"
      And the response reports that the invitation does not exist

    Scenario: Workspace owner reinvites after cancelling a pending invitation
      Given Workspace Owner cancelled a pending invitation for "alice@example.com" to Workspace "Acme"
      When Workspace Owner reinvites "alice@example.com"
      Then "alice@example.com" receives a new invitation verification code
      And Workspace "Acme" has a pending invitation for "alice@example.com"

    Scenario: Workspace owner cannot invite an existing member
      Given User Login "alice@example.com" is a member of Workspace "Acme"
      When Workspace Owner invites "alice@example.com" to Workspace "Acme"
      Then the response status is "400 Bad Request"

  Rule: Accepting invitations

    Scenario Outline: User accepts an invitation addressed to an authenticated identifier
      Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
      And Workspace "Acme" has an invitation for "<recipient>"
      When User Login "jsmith@example.com" submits the invitation verification code
      Then User Login "jsmith@example.com" becomes a member of Workspace "Acme"
      And the membership identifies canonical User Login "jsmith@example.com"

      Examples:
        | recipient           |
        | jsmith@example.com  |
        | j.smith@example.com |

    Scenario: User cannot accept an invitation addressed to another identity
      Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
      And Workspace "Acme" has an invitation for "other@example.com"
      When User Login "jsmith@example.com" submits the invitation verification code
      Then the response status is "400 Bad Request"
      And User Login "jsmith@example.com" is not a member of Workspace "Acme"

    Scenario Outline: User cannot accept an unusable invitation
      Given Workspace "Acme" has an invitation for User Login "alice@example.com" that <condition>
      When User Login "alice@example.com" submits the invitation verification code
      Then the response status is "400 Bad Request"
      And User Login "alice@example.com" is not a member of Workspace "Acme"

      Examples:
        | condition                         |
        | is expired                        |
        | has a different verification code |
        | was cancelled                     |

    Scenario Outline: Existing member replaces the controlling invitation through another authenticated identifier
      Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
      And User Login "jsmith@example.com" joined Workspace "Acme" through an invitation for "<previous recipient>" with Role "app1pkg.LimitedAccessRole"
      And Workspace "Acme" has an invitation for "<new recipient>" with Role "app1pkg.SpecialAPITokenRole"
      When User Login "jsmith@example.com" submits the new invitation verification code
      Then User Login "jsmith@example.com" remains an active member of Workspace "Acme"
      And Workspace "Acme" has exactly one membership for User Login "jsmith@example.com"
      And the invitation for "<previous recipient>" is cancelled
      And Workspace "Acme" has exactly one joined invitation for the membership, addressed to "<new recipient>"
      And User Login "jsmith@example.com" has Role "app1pkg.SpecialAPITokenRole" in Workspace "Acme"

      Examples:
        | previous recipient  | new recipient       |
        | jsmith@example.com  | j.smith@example.com |
        | j.smith@example.com | jsmith@example.com  |

    Scenario Outline: Workspace owner cannot manage a retired invitation
      Given User Login "jsmith@example.com" is an active member of Workspace "Acme" through a joined invitation for "j.smith@example.com" with Role "app1pkg.SpecialAPITokenRole"
      And the previous invitation for "jsmith@example.com" was retired after replacement
      When Workspace Owner <operation>
      Then the response status is "400 Bad Request"
      And User Login "jsmith@example.com" remains an active member of Workspace "Acme"
      And User Login "jsmith@example.com" has Role "app1pkg.SpecialAPITokenRole" in Workspace "Acme"
      And the invitation for "j.smith@example.com" remains joined

      Examples:
        | operation                                                                            |
        | cancels the retired invitation                                                       |
        | updates the retired invitation to Role "app1pkg.LimitedAccessRole"                   |

    Scenario: User cannot replace a membership whose controlling invitation cannot be identified
      Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
      And User Login "jsmith@example.com" is an active member of Workspace "Acme" with Role "app1pkg.LimitedAccessRole"
      And the membership has no identifiable previous controlling invitation
      And Workspace "Acme" has a pending invitation for "j.smith@example.com" with Role "app1pkg.SpecialAPITokenRole"
      When User Login "jsmith@example.com" submits the pending invitation verification code
      Then the response status is "409 Conflict"
      And error message is "A workspace membership is already active for canonical login \"jsmith@example.com\". The existing accepted invitation must be cancelled manually before another invitation can be accepted."
      And User Login "jsmith@example.com" remains an active member of Workspace "Acme" with Role "app1pkg.LimitedAccessRole"
      And the invitation for "j.smith@example.com" remains pending

  Rule: Managing member roles

    Scenario: Workspace owner updates an invited member's roles
      Given User Login "alice@example.com" joined Workspace "Acme" with Role "app1pkg.LimitedAccessRole"
      When Workspace Owner updates the membership to Role "app1pkg.SpecialAPITokenRole"
      Then User Login "alice@example.com" has Role "app1pkg.SpecialAPITokenRole" in Workspace "Acme"
      And the user's joined-workspace record has Role "app1pkg.SpecialAPITokenRole"
      And "alice@example.com" receives a role-update email

  Rule: Ending and restoring membership

    Scenario Outline: Workspace membership ends
      Given User Login "alice@example.com" is a member of Workspace "Acme"
      When <action>
      Then User Login "alice@example.com" is not an active member of Workspace "Acme"
      And the user's joined-workspace record is inactive

      Examples:
        | action                                                           |
        | Workspace Owner removes User Login "alice@example.com"           |
        | User Login "alice@example.com" leaves Workspace "Acme"           |

    Scenario Outline: Previous member accepts a new invitation
      Given User Login "alice@example.com" previously <membership end> Workspace "Acme"
      When Workspace Owner reinvites User Login "alice@example.com"
      And User Login "alice@example.com" accepts the new invitation
      Then User Login "alice@example.com" is an active member of Workspace "Acme"
      And Workspace "Acme" has exactly one membership for User Login "alice@example.com"

      Examples:
        | membership end             |
        | was removed from           |
        | left                       |

  Rule: Invitation validation

    Scenario Outline: Workspace owner cannot invite a malformed email address
      When Workspace Owner invites "<email>" to Workspace "Acme"
      Then the response status is "400 Bad Request"

      Examples:
        | email |
        | a     |
        | bad@  |
        | @bad  |

    Scenario Outline: Workspace owner cannot send an invitation with an invalid role set
      When Workspace Owner sends an invitation with Roles "<roles>"
      Then the response status is "400 Bad Request"

      Examples:
        | roles                                                     |
        |                                                           |
        | not-a-qname                                               |
        | sys.WorkspaceOwner                                        |
        | app1pkg.NonExistentRole                                   |
        | app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole       |

    Scenario Outline: Workspace owner cannot update an invitation with an invalid role set
      Given Workspace "Acme" has a pending invitation for "alice@example.com"
      When Workspace Owner updates the invitation with Roles "<roles>"
      Then the response status is "400 Bad Request"

      Examples:
        | roles                                                     |
        |                                                           |
        | not-a-qname                                               |
        | sys.WorkspaceOwner                                        |
        | app1pkg.NonExistentRole                                   |
        | app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole       |
