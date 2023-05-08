# ctool Technical Design

## Principles

- cluster.json/version must be equal to ctool version, except `upgrade` command 


## cluster.json: Successful deployment SE

Example of the fragment `cluster.json` with comments.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```txt
{
  // Cluster edition type
  "Edition": "SE", 
  
  // ctool version
  "CToolVersion": "0.0.1", 
  
  // List of data centers
  // Exists only for multi-dc deployment
  "DataCenters": [ 
    "dc1", // relates to Nodes[2]
    "dc2", // relates to Nodes[3]
    "dc3"  // relates to Nodes[4]
  ],
  
  // Skipped if empty
  "LastAttemptInfo": "some info about cluster",

  // Cluster nodes
  "Nodes": [
    {
      "NodeRole": "SENode", // Node role

      "Address": "5.255.255.55", // IP address

      // Cluster status
      "State": {
        
        // Version from the last successful attempt
        // Skipped if empty
        "NodeVersion": "0.0.1",
        
        "AttemptNo": 1,  // Attempt number
        
        // Skipped if empty
        "Info": "some info about node"
      }
    }
  ]
}
```

## cluster.json: Fail deployment SE

Example of the fragment `cluster.json` with comments.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```txt
{
  // IP address
  "EditionType": "SE", 
  
  // ctool version
  "ClusterVersion": "0.0.2", 

  // List of data centers
  // Exists only for multi-dc deployment
  "DataCenters": [ 
    "dc1", // relates to Nodes[2]
    "dc2", // relates to Nodes[3]
    "dc3"  // relates to Nodes[4]
  ],
  
  // Skipped if empty
  "LastAttemptError": "some error",
  
  "Nodes": [
    {
      // Node role
      "NodeRole": "SENode",

      // IP address
      "Address": "5.255.255.55",  

      // Cluster status
      "State": {
        // Version from the last successful attempt
        // Skipped if empty
        "NodeVersion": "0.0.1",
        
        // Attempt number
        "AttemptNo": 2,
        
        // Error on the last attempt
        // Skipped if empty
        "Error": "some error"
      }
    },
  ]
}
```

## cluster.json: Replace

Example of the fragment `cluster.json` with comments.

In the full version of `cluster.json` array `Nodes` contains 5 elements.

```txt
{
  // IP address
  "EditionType": "SE", 
  
  // ctool version
  "ClusterVersion": "0.0.2", 

  // List of data centers
  // Exists only for multi-dc deployment
  "DataCenters": [ 
    "dc1", // relates to Nodes[2]
    "dc2", // relates to Nodes[3]
    "dc3"  // relates to Nodes[4]
  ],
  
  // Skipped if empty
  "LastAttemptError": "some error",
  
  "Nodes": [
    {
      // Node role
      "NodeRole": "SENode",

      "State": {    
        "Address": "5.255.255.55",
      }
      "PreviousState": {
      
        "Address": "5.255.255.54",  
      
        // Version from the last successful attempt
        // Skipped if empty
        "NodeVersion": "0.0.1",
        
        // Attempt number
        "AttemptNo": 2,
        
        // Error on the last attempt
        // Skipped if empty
        "Error": "some error"
      }      
    },
  ]
}
```

