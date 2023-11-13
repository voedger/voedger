/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Update the cluster version to the current one",
		RunE:  upgrade,
	}
}

// versions compare (version format: 0.0.1 or 0.0.1-alfa)
// return 1  if version1 > version2
// return -1 if version1 < version2
// return 0 if version1 = version2
func compareVersions(version1 string, version2 string) int {
	v1Components := strings.Split(version1, ".")
	v2Components := strings.Split(version2, ".")

	for i := 0; i < len(v1Components) || i < len(v2Components); i++ {
		v1 := 0
		v2 := 0

		if i < len(v1Components) {
			v1 = parseVersionComponent(v1Components[i])
		}
		if i < len(v2Components) {
			v2 = parseVersionComponent(v2Components[i])
		}

		if v1 > v2 {
			return 1
		} else if v1 < v2 {
			return -1
		}
	}

	return 0
}

func parseVersionComponent(component string) int {
	if strings.Contains(component, "-") {
		component = strings.Split(component, "-")[0]
	}
	version, _ := fmt.Sscanf(component, "%d", &version)
	return version
}

func needUpgrade(cluster *clusterType) bool {
	return cluster.ActualClusterVersion != "" && compareVersions(version, cluster.ActualClusterVersion) == 1
}

func upgrade(cmd *cobra.Command, arg []string) error {

	cluster, err := newCluster()
	if err != nil {
		return err
	}

	if cluster.ActualClusterVersion == cluster.DesiredClusterVersion {
		fmt.Println("no update required")
		return nil
	}

	err = mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	return nil
}
