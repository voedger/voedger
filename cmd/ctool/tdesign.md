# ctool Technical Design

## Principles

- cluster.json/version must be equal to ctool version, except `upgrade` command 


## cluster.json: Successful deployment SE

Example of the fragment `cluster.json` with comments.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```txt
{
  // Edition
  "EditionType": "SE", 
  
  // ctool version
  "ActualClusterVersion": "0.0.2",

  // Skipped if empty
  "LastAttemptInfo": "some info",

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

```txt
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

      // Attempt number
      "AttemptNo": 2,
        
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

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```txt
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
