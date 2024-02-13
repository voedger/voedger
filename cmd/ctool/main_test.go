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

	"github.com/fatih/color"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/logger"
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
	require.NoError(err)

	c = failSECluster()
	err = c.saveToJSON()
	require.NoError(err)

	c = successCECluster()
	err = c.saveToJSON()
	require.NoError(err)

	c = failCECluster()
	err = c.saveToJSON()
	require.NoError(err)

}

// tests ctool commands
func TestCtoolCommands(t *testing.T) {
	require := require.New(t)

	red = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	logger.PrintLine = printLogLine
	prepareScripts()
	defer func() {
		err := deleteScriptsTempDir()
		if err != nil {
			loggerError(err.Error())
		}
	}()

	version = "0.0.1"
	deleteDryRunDir()
	err := deleteClusterJson()
	require.NoError(err)

	defer deleteDryRunDir()

	// Version command is performed without error
	err = execRootCmd([]string{"./ctool", "version", "--dry-run"}, version)
	require.NoError(err)

	// the command validate return the error because the configuration of the cluster has not yet been created
	err = execRootCmd([]string{"./ctool", "validate", "--dry-run"}, version)
	require.Error(err)

	// execute the init command
	err = execRootCmd([]string{"./ctool", "init", "SE", "10.0.0.21", "10.0.0.22", "10.0.0.23", "10.0.0.24", "10.0.0.25", "--dry-run", "--ssh-key", "key", "--acme-domain", "domain1,domain2,domain3"}, version)
	require.NoError(err)

	dryRun = true
	cluster := newCluster()
	require.Equal("domain1,domain2,domain3", cluster.Acme.domains())

	// repeat command init should give an error
	err = execRootCmd([]string{"./ctool", "init", "SE", "10.0.0.21", "10.0.0.22", "10.0.0.23", "10.0.0.24", "10.0.0.25", "--dry-run", "--ssh-key", "key"}, version)
	require.Error(err)

	// execute the replace command
	err = execRootCmd([]string{"./ctool", "replace", "db-node-1", "10.0.0.28", "--dry-run", "--ssh-key", "key"}, version)
	require.NoError(err)

	// replace node to the address from the list of Replacedaddresses should give an error
	err = execRootCmd([]string{"./ctool", "replace", "10.0.0.28", "10.0.0.23", "--dry-run", "--ssh-key", "key"}, version)
	require.Error(err)

	// upgrade without changing the ctool version should give an error
	err = execRootCmd([]string{"./ctool", "upgrade", "--dry-run", "--ssh-key", "key"}, version)
	require.Error(err)

	// increase the ctool version.Upgrade is performed without error
	version = "0.0.2"
	err = execRootCmd([]string{"./ctool", "upgrade", "--dry-run", "--ssh-key", "key"}, version)
	require.NoError(err)

	// after a successful upgrade, command validate should work without errors
	err = execRootCmd([]string{"./ctool", "validate", "--dry-run"}, version)
	require.NoError(err)

	// понижаем версию ctool. Команда upgrade должна выдать ошибку
	version = "0.0.1"
	err = execRootCmd([]string{"./ctool", "upgrade", "--dry-run", "--ssh-key", "key"}, version)
	require.Error(err)
}

func TestAcmeDomains(t *testing.T) {
	require := require.New(t)

	cluster := newCluster()
	require.Equal("", cluster.Acme.domains())

	cluster.Acme.Domains = []string{"domain1.io", "domain2.io"}
	require.Equal("domain1.io,domain2.io", cluster.Acme.domains())

	cluster.Acme.addDomains("domain2.io,domain3,domain4")
	require.Equal("domain1.io,domain2.io,domain3,domain4", cluster.Acme.domains())

	cluster.Acme.removeDomains("domain2.io,domain4")
	require.Equal("domain1.io,domain3", cluster.Acme.domains())
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
	require.NoError(err)

	defer func() {
		err := deleteScriptsTempDir()
		require.NoError(err)
	}()

	err = ioutil.WriteFile(filepath.Join(scriptsTempDir, "test-script.sh"), []byte(script), 0700)
	require.NoError(err)

	err = os.Setenv("TEST_VAR", "test_value")
	require.NoError(err)

	err = newScriptExecuter("", "").run("test-script.sh")
	require.NoError(err)

	err = os.Setenv("TEST_VAR", "new_test_value")
	require.NoError(err)

	err = newScriptExecuter("", "").run("test-script.sh")
	require.Error(err)
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
	if _, err := os.Stat(dryRunDir); os.IsNotExist(err) {
		return nil
	}

	err := os.RemoveAll(dryRunDir)
	if err != nil {
		return err
	}

	return nil
}
