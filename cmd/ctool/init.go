/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

var (
	initCECmd, initSECmd *cobra.Command
)
var skipStacks []string

func newInitCmd() *cobra.Command {
	initCECmd = &cobra.Command{
		Use:   "CE [<ipaddr>...]",
		Short: "Creates the file cluster.json for the CE edition cluster",
		Args:  cobra.MaximumNArgs(1),
		RunE:  initCE,
	}

	initSECmd = &cobra.Command{
		Use:   "SE [<ipaddr>...]",
		Short: "Creates the file cluster.json for the SE edition cluster",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != initSeArgCount {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: initSE,
	}

	initSECmd.Flags().StringSliceVar(&skipStacks, "skip-stack", []string{}, "Specify docker compose stacks to skip")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Creates the file cluster.json for cluster",
	}

	initCmd.AddCommand(initCECmd, initSECmd)

	return initCmd

}

// nolint
func parseIpArg(arg string) (resArg string, err error) {
	if net.ParseIP(arg) == nil {
		return "", errors.New("invalid IP address " + arg)
	}

	return arg, nil
}

// nolint
func parseDeployArgs(args []string) error {
	if len(args) == 0 {
		return errors.New("the list of command arguments is empty")
	}

	var err error

	if args[0] == "SE" {
		if len(args) != initSeArgCount {
			return errors.New("invalid number of arguments")
		}

		for i := deploySeFirstNodeArgIdx; i < deploySeFirstNodeArgIdx+seNodeCount; i++ {
			_, err = parseIpArg(args[i])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// nolint
func initCE(cmd *cobra.Command, args []string) error {

	cluster, err := newCluster()
	if err != nil {
		return err
	}

	defer func(cluster *clusterType) {
		err := cluster.saveToJSON()
		if err != nil {
			logger.Error(err.Error())
		}
	}(cluster)

	if !cluster.Draft {
		return ErrClusterConfAlreadyExists
	}

	c := newCmd(ckInit, "CE "+strings.Join(args, " "))
	if err = cluster.applyCmd(c); err != nil {
		logger.Error(err.Error())
		return err
	}

	err = mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	err = cluster.readFromInitArgs(cmd, args)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	err = cluster.validate()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("Cluster configuration is ok")

	return err
}

// nolint
func initSE(cmd *cobra.Command, args []string) error {

	cluster, err := newCluster()
	if err != nil {
		return err
	}

	if !cluster.Draft {
		return ErrClusterConfAlreadyExists
	}

	c := newCmd(ckInit, "SE "+strings.Join(args, " "))
	c.SkipStacks = skipStacks
	if err := cluster.applyCmd(c); err != nil {
		logger.Error(err.Error())
		return err
	}

	defer func(cluster *clusterType) {
		err := cluster.saveToJSON()
		if err != nil {
			logger.Error(err.Error())
		}
	}(cluster)
	err = mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	err = cluster.readFromInitArgs(cmd, args)
	if err != nil {
		return err
	}

	err = cluster.validate()
	if err == nil {
		println("cluster configuration is ok")
		if err = cluster.Cmd.apply(cluster); err != nil {
			logger.Error(err)
		}
	}

	return err
}
