# Ideal voedger arcitecture

## Components

## Software components

Core:

- `httpadapter`: sends http(s) requests to voedger
- `router`: routes requests to appropriate processors
- `procs`: manage processors

Processors:

- `cp`: command processor
- `qp`: query processor
- `bp`: blob processor
- `np`: notification processor
- `jp`: job processor
- `appmgr`: application management processor
  - Deploy/undeply app to cluster
- `apartmgr`: application partition management processor
  - Deploy/undeply apart to `procs`
- `wsmgr`: workspace manager processor
  - query/create/deactive workspaces
  - Used by other processors

## Hardware components

- federation
- cluster
- One or more nodes

## Scenarios

### Overview

- bootstart
  - output: working cluster app
- deploy app
  - input: appdef, deployment descriptor
- deploy apart
  - prerequisites
    - app with given version is deployed
    - apart with the same number may be deployed
  - input: appdef, deployment descriptor, number of apart
  - output: 
