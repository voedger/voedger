/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var testVersion string = "0.0.1-dummy"

func successSECluster() clusterType {
	return clusterType{
		Edition:               clusterEditionSE,
		DesiredClusterVersion: version,
		ActualClusterVersion:  version,
		Nodes: []nodeType{
			{
				NodeRole: "SENode",
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: version,
				},
			},
			{
				NodeRole: "SENode",
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: version,
				},
			},
			{
				NodeRole: "DBNode",
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.57",
					NodeVersion: version,
				},
			},
			{
				NodeRole: "DBNode",
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: version,
				},
			},
			{
				NodeRole: "DBNode",
				ActualNodeState: &nodeStateType{
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
		Nodes: []nodeType{
			{
				NodeRole: "SENode",
				DesiredNodeState: &nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: version,
				},
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.55",
					NodeVersion: "",
				},
			},
			{
				NodeRole: "SENode",
				DesiredNodeState: &nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: version,
				},
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.56",
					NodeVersion: "",
				},
			},
			{
				NodeRole: "DBNode",
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.57",
					NodeVersion: version,
				},
			},
			{
				NodeRole: "DBNode",
				Error:    "error information on the node",
				DesiredNodeState: &nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: version,
				},
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.58",
					NodeVersion: nodeFailVersion,
				},
			},
			{
				NodeRole: "DBNode",
				ActualNodeState: &nodeStateType{
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
		Nodes: []nodeType{
			{
				NodeRole: "CENode",
				ActualNodeState: &nodeStateType{
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
				NodeRole: "CENode",
				Error:    "error information on the node",
				DesiredNodeState: &nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: version,
				},
				ActualNodeState: &nodeStateType{
					Address:     "5.255.255.59",
					NodeVersion: "",
				},
			},
		},
	}
}

func TestClusterJSON(t *testing.T) {

	// FIXME //TODO
	t.Skip("not implemented yet")

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

// tests ctool commands
func TestCtoolCommands(t *testing.T) {
	require := require.New(t)

	deleteDryRunDir()
	err := deleteClusterJson()
	require.NoError(err, err)

	defer deleteDryRunDir()

	// execute the init command
	err = execRootCmd([]string{"./ctool", "init", "SE", "10.0.0.21", "10.0.0.22", "10.0.0.23", "10.0.0.24", "10.0.0.25", "--dry-run", "--ssh-key", "key"}, version)
	require.NoError(err, err)

	// repeat command init should give an error
	err = execRootCmd([]string{"./ctool", "init", "SE", "10.0.0.21", "10.0.0.22", "10.0.0.23", "10.0.0.24", "10.0.0.25", "--dry-run", "--ssh-key", "key"}, version)
	require.Error(err, err)

	// execute the replace command
	err = execRootCmd([]string{"./ctool", "replace", "db-node-1", "10.0.0.28", "--dry-run", "--ssh-key", "key"}, version)
	require.NoError(err, err)

	// replace node to the address from the list of Replacedaddresses should give an error
	err = execRootCmd([]string{"./ctool", "replace", "10.0.0.28", "10.0.0.23", "--dry-run", "--ssh-key", "key"}, version)
	require.Error(err, err)
}

// Testing the availability of the variable environment from scripts caused by PipedExec
func TestVariableEnvironment(t *testing.T) {
	require := require.New(t)

	dryRun = true

	script := `#!/usr/bin/env bash
set -euo pipefail
set -x
echo "TEST_VAR = $TEST_VAR"

if [ "$TEST_VAR" != "test_value" ]; then
  exit 1
fi
`
	err := createScriptsTempDir()
	require.NoError(err, err)

	defer func() {
		err := deleteScriptsTempDir()
		require.NoError(err, err)
	}()

	err = ioutil.WriteFile(filepath.Join(scriptsTempDir, "test-script.sh"), []byte(script), 0700)
	require.NoError(err, err)

	err = os.Setenv("TEST_VAR", "test_value")
	require.NoError(err, err)

	err = newScriptExecuter("", "").run("test-script.sh")
	require.NoError(err, err)

	err = os.Setenv("TEST_VAR", "new_test_value")
	require.NoError(err, err)

	err = newScriptExecuter("", "").run("test-script.sh")
	require.Error(err, err)
}

func deleteClusterJson() error {
	fname := "cluster.json"
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		return nil
	}

	err := os.Remove(fname)
	if err != nil {
		return err
	}

	return nil
}

func deleteDryRunDir() error {
	dryRunDir := "dry-run"
	if _, err := os.Stat(dryRunDir); os.IsNotExist(err) {
		return nil
	}

	err := os.RemoveAll(dryRunDir)
	if err != nil {
		return err
	}

	return nil
}
