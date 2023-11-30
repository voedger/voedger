# ctool Technical Design

## Principles

- cluster.json/version must be equal to ctool version, except `upgrade` command 

## Generalized example сluster.json

```javascript
{
// Edition (type of editorial office)
  "EditionType": "SE", 

  // Current version of CTool
  "ActualClusterVersion": "0.0.2",

  // The desired version of CTool (used for the UPGRADE command)
  "DesiredClusterVersion": "0.0.3",

  // The team that needs to be performed
  "Cmd": {
    "Kind": "command kind (name)",
    "Args": "command arguments separated by a space"
  },

  // Error of the last attempt (present only if the command is unsuccessful)
  "LastAttemptError": "some error",

  // List of cluster components
  "Nodes": [
    {
      // The role of the node
      "NodeRole": "SENode",

     // Information about the last attempt (transmitted if it is empty)
      "Info": "some info",

      // Actual state of the node
      "ActualNodeState": {
        "Address": "5.255.255.55",
        "NodeVersion": "0.0.2"
      },

      // Error of the last attempt (transmitted if it is empty)
      "Error": "some error",

     // desired state of the node
      "DesiredNodeState": {
        "Address": "5.255.255.55",
        "NodeVersion": "0.0.2"
      }
    },
  ],

  // List of replaced addresses (present only when replacing nodes)
  "ReplacedAddresses": [
    "10.0.0.28",
    "10.0.0.29"
  ]
}
```

## cluster.json: Successful deployment SE

Example of the fragment `cluster.json` with comments.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```javascript
{
  // Edition
  "EditionType": "SE", 
  
  // ctool version
  "ActualClusterVersion": "0.0.2",

  "Nodes": [
    {
      // Node role
      "NodeRole": "SENode",
   
      // info on the last attempt
      // Skipped if empty
      "Info": "some info"
        
      // actual node state     
      "ActualNodeState": {
        "Address": "5.255.255.55",
        "NodeVersion": "0.0.2"
      }

    },
  ]
}
```

## cluster.json: Fail deployment SE

Example of the fragment `cluster.json` with comments.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```javascript
{
  // Edition
  "EditionType": "SE", 
  
  // ctool version
  "ActualClusterVersion": "",
  "DesiredClusterVersion": "0.0.2",

  // actual command for apply
  "Cmd": {
    "Kind": "command kind (name)",
    "Args": "command arguments separated by a space"
  },

  // Skipped if empty
  "LastAttemptError": "some error",

  // List of data centers
  // Exists only for multi-dc deployment
  "DataCenters": [ 
    "dc1", // relates to Nodes[2]
    "dc2", // relates to Nodes[3]
    "dc3"  // relates to Nodes[4]
  ],

  "Nodes": [
    {
      // Node role
      "NodeRole": "SENode",

       
      // Error on the last attempt
      // Skipped if empty
      "Error": "some error"

      "ActualNodeState": {
        "Address": "",
        "NodeVersion": ""
      }
      "DesiredNodeState": {
        "Address": "5.255.255.55",
        "NodeVersion": "0.0.2"
      }

    },
  ]
}
```

## cluster.json: Replace

Example of the fragment `cluster.json` with comments.
Replace DBNode from 10.0.0.21 to 10.0.0.22

In case of successful completion of the `Replace` command, the replaced address is added to the Replacedaddresses list and stored in` Cluster.json`.Re -use in the address of the address from the list `Replacedaddresses` is unacceptable.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```javascript
{
  // Edition
  "EditionType": "SE", 
  
  // ctool version
  "ActualClusterVersion": "0.0.2",
  
  // replace command for apply
  "Cmd": {
    "Kind": "replace",
    "Args": "10.0.0.21 10.0.0.22"
  },

  "Nodes": [
    {
      // Node role
      "NodeRole": "DBNode",
       
      "ActualNodeState": {
        "Address": "10.0.0.21",
        "NodeVersion": "0.0.2"
      }
      "DesiredNodeState": {
        "Address": "10.0.0.22",
        "NodeVersion": "0.0.2"
      }

    },
  ],
  "ReplacedAddresses": [
    "10.0.0.28",
    "10.0.0.29"
  ]
}
```
## NodeControllerFunction

Works if: section DesiredNodeState exists

  - Optionally update docker
  - Assign ActualNodeState = DesiredNodeState
  - Remove DesiredNodeState

## ClusterCntrollerFunction

Works if: Cmd exists 

  - DeploySwarm()
  - UpdateSEDockerStack()
  - UpdateDBScyllaDockerStack()
  - UpdateMonDockerStack()
  - Assign ActualClusterVersion = DesiredClusterVersion
  - Remove Cmd


## cluster.json: Upgrade


- the ctool version is linked from the `version` file;
- when running ctool, the cluster structure is read from `cluster.json` for comparing the weights of `ctool`, `ActualClusterVersion` and `DesiredClusterVersion`;
- if in `cluster.json` has an incomplete command and `DesiredClusterVersion` differs from the version of the `ctool`, then the work of the ctool stops with a message about the need to install the ctool version of the `DesiredClusterVersion` to complete the command;
- if in `cluster.json` `ActualClusterVersion` is older than the version of the `ctool`, then the work of the ctool stops with a message about the need to install the ctool version of the `ActualClusterVersion` to continue working;
- if in `cluster.json` `ActualClusterVersion` is not empty and it is younger than the `ctool` version, then a message is displayed about the need to execute the `upgrade` command. Any other commands that change the cluster configuration can be executed only after the `upgrade` command is successfully completed;
- the algorithm for extracting the `upgrade` command is similar to the algorithm of the `init` command. `Upgrade` deploys the cluster on top of the previously deployed one. This operation reinstalls, if necessary, docker stacks. User data is not affected.  


Example of the fragment `cluster.json` with `upgrade` command.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```javascript
{
  // Edition
  "EditionType": "SE", 
  
  // ctool version
  "ActualClusterVersion": "0.0.2",
  "DesiredClusterVersion": "0.0.3",

  // upgrade command
  "Cmd": {
    "Kind": "upgrade",
    "Args": ""
  },

  "Nodes": [
    {
      // Node role
      "NodeRole": "SENode",

      "ActualNodeState": {
        "Address": "5.255.255.55",
        "NodeVersion": "0.0.2"
      }
      "DesiredNodeState": {
        "Address": "5.255.255.55",
        "NodeVersion": "0.0.3"
      }

    },
  ]
}
```