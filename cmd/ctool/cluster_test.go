/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var testVersion string = "0.0.1-dummy"

func successSECluster() clusterType {
	return clusterType{
		Edition:               clusterEditionSE,
		DesiredClusterVersion: version,
		ActualClusterVersion:  version,
		LastAttemptInfo:       "some info about cluster",
		DataCenters: []string{
			"dc1", "dc2", "dc3",
		},
		Nodes: []nodeType{
			{
				NodeRole:  "SENode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: version,
				},
			},
			{
				NodeRole:  "SENode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: version,
				},
			},
			{
				NodeRole:  "DBNode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.57",
					NodeVersion: version,
				},
			},
			{
				NodeRole:  "DBNode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: version,
				},
			},
			{
				NodeRole:  "DBNode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
				},
			},
		},
	}
}

func failSECluster() clusterType {
	return clusterType{
		Edition:               clusterEditionSE,
		ActualClusterVersion:  "",
		DesiredClusterVersion: version,
		LastAttemptError:      "some error",
		DataCenters: []string{
			"dc1", "dc2", "dc3",
		},
		Nodes: []nodeType{
			{
				NodeRole:  "SENode",
				AttemptNo: 1,
				Info:      "some info about node",
				DesiredNodeState: nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: version,
				},
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: "",
				},
			},
			{
				NodeRole:  "SENode",
				AttemptNo: 2,
				Info:      "error information on the node",
				DesiredNodeState: nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: version,
				},
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: "",
				},
			},
			{
				NodeRole:  "DBNode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.57",
					NodeVersion: version,
				},
			},
			{
				NodeRole:  "DBNode",
				AttemptNo: 1,
				Error:     "error information on the node",
				DesiredNodeState: nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: version,
				},
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: nodeFailVersion,
				},
			},
			{
				NodeRole:  "DBNode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
				},
			},
		},
	}
}

func successCECluster() clusterType {
	return clusterType{
		Edition:               clusterEditionCE,
		DesiredClusterVersion: version,
		ActualClusterVersion:  version,
		LastAttemptInfo:       "some info about cluster",
		Nodes: []nodeType{
			{
				NodeRole:  "CENode",
				AttemptNo: 1,
				Info:      "some info about node",
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
				},
			},
		},
	}
}

func failCECluster() clusterType {
	return clusterType{
		Edition:               clusterEditionCE,
		DesiredClusterVersion: version,
		ActualClusterVersion:  version,
		LastAttemptError:      "some error",
		Nodes: []nodeType{
			{
				NodeRole:  "CENode",
				AttemptNo: 1,
				Error:     "error information on the node",
				DesiredNodeState: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
				},
				ActualNodeState: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: "",
				},
			},
		},
	}
}

func TestClusterJSON(t *testing.T) {
	version = testVersion
	require := require.New(t)

	c := successSECluster()
	err := c.saveToJSON()
	require.NoError(err, err)

	c = failSECluster()
	err = c.saveToJSON()
	require.NoError(err, err)

	c = successCECluster()
	err = c.saveToJSON()
	require.NoError(err, err)

	c = failCECluster()
	err = c.saveToJSON()
	require.NoError(err, err)

}
