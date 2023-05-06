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
		Edition:         clusterEditionSE,
		CToolVersion:    version,
		LastAttemptInfo: "some info about cluster",
		DataCenters: []string{
			"dc1", "dc2", "dc3",
		},
		Nodes: []nodeType{
			{
				NodeRole: "SENode",
				State: nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
			{
				NodeRole: "SENode",
				State: nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
			{
				NodeRole: "DBNode",
				State: nodeStateType{
					Address:     "5.255.255.57",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
			{
				NodeRole: "DBNode",
				State: nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
			{
				NodeRole: "DBNode",
				State: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
		},
	}
}

func failSECluster() clusterType {
	return clusterType{
		Edition:          clusterEditionSE,
		CToolVersion:     version,
		LastAttemptError: "some error",
		DataCenters: []string{
			"dc1", "dc2", "dc3",
		},
		Nodes: []nodeType{
			{
				NodeRole: "SENode",
				State: nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
			{
				NodeRole: "SENode",
				State: nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: "",
					AttemptNo:   2,
					Info:        "error information on the node",
				},
			},
			{
				NodeRole: "DBNode",
				State: nodeStateType{
					Address:     "5.255.255.57",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
			{
				NodeRole: "DBNode",
				State: nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: nodeFailVersion,
					AttemptNo:   1,
					Error:       "error information on the node",
				},
			},
			{
				NodeRole: "DBNode",
				State: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
		},
	}
}

func successCECluster() clusterType {
	return clusterType{
		Edition:         clusterEditionCE,
		CToolVersion:    version,
		LastAttemptInfo: "some info about cluster",
		Nodes: []nodeType{
			{
				NodeRole: "CENode",
				State: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
					AttemptNo:   1,
					Info:        "some info about node",
				},
			},
		},
	}
}

func failCECluster() clusterType {
	return clusterType{
		Edition:          clusterEditionCE,
		CToolVersion:     version,
		LastAttemptError: "some error",
		Nodes: []nodeType{
			{
				NodeRole: "CENode",
				State: nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: "",
					AttemptNo:   1,
					Error:       "error information on the node",
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
