# Bus

Bus connects federation components

# Routing

## Selecting Cluster

- Request comes to `Federation Entry Point` using technology like `Amazon Geolocation routing policy`
- If destination cluster is not current one request is followed to destination cluster
- Otherwise application routing is started

## Selecting Role

- Request is sent to application Main role
- Main role can return `Redirect` answer
  - Request is sent to role defined by Redirect
  - Role can also return `Redirect` answer
  - Number of hops is limited to 4
  
## Selecting Partition

- By default partition from request is used
- If partition is not available `Emergency Partition` is used
- If `Emergency Partition` is not defined yet it is selected on random basis

# Bus Components

- Each cluster has few `Routers` (number is fixed so far)
- Each router has `Internal` and `External` addresses
  - `Internal Address` is used for task-router connections
  - `External Address` is used for router-router connections
- Each `task` is connected to all routers by websocket
- Each router keeps a connection with other clusters and forward requests if needed

# DNS

- `Federation Entry Point`: heeus.cloud
  - Resolved by `Amazon Geolocation routing policy`
- `Master Cluster`: master.master.heeus.cloud
- `Worker Clusters`
  - `<cluster>.<region>.<federation-entry-point>`
    - spb1.ru.heeus.cloud
    - spb2.ru.heeus.cloud

# Request

- `<federation-entry-point>/<region>/<cluster>/<user>/<app>/<service>/<wsid>/<module>/<function>`
  - `spb1.ru.heeus.cloud/api/ru/spb1/<user>/<app>/<service>/<wsid>/<module>/<function>`

# Authentication

# Authenticating Service Task

- When Service `Task` starts it gets Service Token signed by `Cluster Key`
- Task connects to `Router` and sends Service Token
- Router also gets (part of) `Cluster Key` and uses it to authenticate task

# Authorizing Client

- Each application has `Auth Service`
- Client sends request to `Auth Service`
- `Auth Service` signs JWT token using `Application Secret`
  - JWT token content includes version of the secret
- Client sends JWT signed by application

# Links

- [Amazon Route 53 — Routing Policies](https://medium.com/tensult/amazon-route-53-routing-policies-cbe356b851d3)