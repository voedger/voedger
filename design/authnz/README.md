### Abstract

Authorization and authentication design

- [Concepts](#concepts)
- [ACL Rules](#acl-rules)
- [Query AuthNZ process](#query-authnz-process)
- [Command AuthNZ process](#command-authnz-process)
- [Detailed design](#detailed-design)

### Concepts (Основные понятия)
Naming based on [AuthNZ: Existing concepts](https://dev.heeus.io/launchpad/#!19546)

- Subject. Entity that can make a request - User/Device/Service
- Login. Represents a subject which can log in (synonym: sign in), user/device
- Profile. Linked to login, personal data and other application specific information
- Principal (Принципал). Unique key which can be used in ACL (список управления доступом)
  - Login | Group | Role
- Role (Роль). Schema-level principal (predefined group, предопределенная группа)
  - Allows to create predefined ACLs (предопределенные списки управления доступом)
  - Examples
    - unTill: Waiter, Waiter+, Manager
    - PK: Executor, Executor+, Manager
- Group (Группа). Workspace-level principal
- PrincipalToken (То́кен Принципала) - token which authenticated principals (токен который удостоверяет подлинность принципалов)
  - Login + Role/Group memberships
- ACL. Acces Control List (список управления доступом)
  - Currently we use predefined ACLs only (предопределенные списки управления доступом)
    - ACL managements too complicated
  - Users can only manage groups and roles membership
  - Permissions for Hosts can be manages by
    - GRANT ROLE ChargeBee TO ADDRESS <ip>

### ACL Rules
- “Principal P from Workspace W is [Allowed/Denied] Operation O on Resources matching ResourcePattern RP”.
  - Principal
  - Policy (Allow/Deny)
  - Operation
  - ResourcePattern
  - MembershipInheritance (00, 10, 11, 01)
  - Ref. comments [here](https://dev.heeus.io/launchpad/#!19546)

### Query AuthNZ process

|Step   |Actor      | Served by   |
|-      |---------- | ----------  |
|Send a request to the QueryProcessor |Subject |
|Authenticate Principal|QueryProcessor |IAuthenticator.Authenticate()
|Authorize EXECUTE operation|QueryProcessor |IAuthorizer.Authorize()
|Opt: Authorize READ operation|QueryProcessor|IAuthorizer.Authorize()

### Command AuthNZ process
|Step|Actor|Served by|
|-|-|-|
|Send a request to the CommandProcessor|Subject |
|Authenticate Principal|CommandProcessor |IAuthenticator.Authenticate()
|Authorize EXECUTE operation|CommandProcessor |IAuthorizer.Authorize()
|Authorize fields CREATE/UPDATE|CommandProcessor |IAuthenticator.Authorize() 

### Detailed design

- [Reset password](reset-password.md)

### Components

- [iauthnz](https://github.com/heeus/core/tree/main/iauthnz)

### See also

- [Originated from A&D: AuthNZ](https://dev.heeus.io/launchpad/#!17808)
- [Slack design: WDocs](https://dev.heeus.io/launchpad/#!19080)
- [AuthNZ: Existing concepts](https://dev.heeus.io/launchpad/#!19546) (including comments!)

