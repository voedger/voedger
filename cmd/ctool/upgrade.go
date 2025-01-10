/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func newUpgradeCmd() *cobra.Command {
	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update the cluster version to the current one",
		RunE:  upgrade,
	}

	c := newCluster()
	if c.Edition != clusterEditionCE && !addSshKeyFlag(upgradeCmd) {
		return nil
	}

	return upgradeCmd

}

// versions compare (version format: 0.0.1 or 0.0.1-alfa)
// return 1  if version1 > version2
// return -1 if version1 < version2
// return 0 if version1 = version2
func compareVersions(version1 string, version2 string) int {
	return semver.Compare("v"+version1, "v"+version2)
}

func upgrade(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
	cluster := newCluster()
	var err error

	ok, e := cluster.needUpgrade()
	if e != nil {
		return e
	}

	if !ok {
		return ErrNoUpdgradeRequired
	}

	err = mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	c := newCmd(ckUpgrade, args)
	defer saveClusterToJson(cluster)

	if err = cluster.applyCmd(c); err != nil {
		return err
	}

	if err = cluster.Cmd.apply(cluster); err != nil {
		return err
	}

	return nil
}
