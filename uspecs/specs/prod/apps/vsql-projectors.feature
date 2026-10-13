Feature: VSQL projector triggers

  VADeveloper declares a projector once when it must run after every command
  in an application.

  Rule: Trigger after all commands

    Background:
      Given the application provides local command "sales.CreateOrder"
      And the application imports command "inventory.ReserveStock"
      And the deployed VSQL schema declares
        """
        PROJECTOR RecordCommand AFTER EXECUTE ON ALL COMMANDS;
        """

    Scenario Outline: Projector runs after each successful command
      When command "<command>" executes successfully
      Then projector "RecordCommand" executes for command "<command>"

      Examples:
        | command                |
        | sales.CreateOrder      |
        | inventory.ReserveStock |
        | sys.CUD                |

    Scenario: Synchronous projector runs after every command
      Given the deployed VSQL schema declares
        """
        SYNC PROJECTOR UpdateImmediately AFTER EXECUTE ON ALL COMMANDS;
        """
      When command "sales.CreateOrder" executes successfully
      Then projector "UpdateImmediately" executes for command "sales.CreateOrder"

  Rule: Existing projector trigger forms

    Background:
      Given the application provides commands "sales.CreateOrder" and "sales.CancelOrder"
      And the deployed VSQL schema declares
        """
        PROJECTOR RecordEveryCommand AFTER EXECUTE ON ALL COMMANDS;
        PROJECTOR RecordCreatedOrder AFTER EXECUTE ON sales.CreateOrder;
        """

    Scenario: Command-specific projector runs for its declared command
      When command "sales.CreateOrder" executes successfully
      Then projector "RecordEveryCommand" executes for command "sales.CreateOrder"
      And projector "RecordCreatedOrder" executes for command "sales.CreateOrder"

    Scenario: Command-specific projector ignores another command
      When command "sales.CancelOrder" executes successfully
      Then projector "RecordEveryCommand" executes for command "sales.CancelOrder"
      And projector "RecordCreatedOrder" does not execute

