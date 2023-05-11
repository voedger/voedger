/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"sync"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

var (
	force      bool
	appCompose string = "gcr.io/cadvisor/cadvisor:latest"
	dbCompose  string = "scylladb/scylla"
)

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Applies the modified configuration to the cluster",
		RunE:  apply,
	}

	cmd.Flags().BoolVarP(&force, "force", "", false, "forced reinstallation of the required environment")
	cmd.Flags().StringVarP(&appCompose, "app-compose", "", appCompose, "name of the application image other than default")
	cmd.Flags().StringVarP(&dbCompose, "db-compose", "", dbCompose, "name of the db server image other than default")

	return cmd
}

func apply(cmd *cobra.Command, arg []string) error {
	cluster := newCluster()
	for _, n := range cluster.Nodes {
		n.cluster = cluster
	}
	defer cluster.saveToJSON()

	var err error

	if err = cluster.validate(); err != nil {
		logger.Error(err.Error)
		return err
	}

	if len(arg) > 0 {
		cluster.sshKey, err = expandPath(arg[0])
		if err != nil {
			return err
		}

	}

	if cluster.Cmd.isEmpty() {
		logger.Info("no active command found to apply")
		return nil
	}

	cluster.Draft = false

	if err := installYq(); err != nil {
		logger.Error(err.Error())
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(cluster.Nodes))

	for i := 0; i < len(cluster.Nodes); i++ {
		cluster.Nodes[i].cluster = cluster
		go func(node *nodeType) {
			defer wg.Done()
			if err := node.nodeControllerFunction(); err != nil {
				logger.Error(err.Error)
			}
		}(&cluster.Nodes[i])
	}

	wg.Wait()

	if cluster.existsNodeError() {
		return ErrorPreparingClusterNodes
	}

	return cluster.clusterControllerFunction()
}

/*
func apply(cmd *cobra.Command, arg []string) error {

	cluster := newCluster()

	err := cluster.validate()
	if err != nil {
		return err
	}

	cluster.Draft = false
	defer cluster.saveToJSON()

	if len(arg) > 0 {
		cluster.sshKey, err = expandPath(arg[0])
		if err != nil {
			return err
		}

	}

	err = mkCommandDirAndLogFile(cmd)
	if err != nil {
		return err
	}

	switch cluster.Edition {
	case clusterEditionCE:
		err = deployCeCluster(cluster)
	case clusterEditionSE:
		err = deploySeCluster(cluster)
	default:
		err = ErrorInvalidClusterEdition
	}

	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}
*/

// Install yq (yaml parser)
func installYq() error {
	prepareScripts("yq-install.sh")
	return newScriptExecuter("", "localhost").run("yq-install.sh")
}
