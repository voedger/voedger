/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"net"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

var (
	initCECmd, initSECmd *cobra.Command
)

func newInitCmd() *cobra.Command {
	initCECmd = &cobra.Command{
		Use:   "CE [<ipaddr>...]",
		Short: "Creates the file cluster.json for the CE edition cluster",
		Args:  cobra.MaximumNArgs(1),
		RunE:  initCE,
	}

	initSECmd = &cobra.Command{
		Use:   "SE [<ipaddr>...] [<data-centers>...]",
		Short: "Creates the file cluster.json for the SE edition cluster",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != initSeArgCount && len(args) != initSeWithDCArgCount {
				return ErrorInvalidNumberOfArguments
			}
			return nil
		},
		RunE: initSE,
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Creates the file cluster.json for cluster",
	}

	initCmd.AddCommand(initCECmd, initSECmd)

	return initCmd

}

func parseIpArg(arg string) (resArg string, err error) {
	if net.ParseIP(arg) == nil {
		return "", errors.New("invalid IP address " + arg)
	}

	return arg, nil
}

func parseDeployArgs(args []string) error {
	if len(args) == 0 {
		return errors.New("the list of command arguments is empty")
	}

	var err error

	if args[0] == "SE" {
		if len(args) != initSeArgCount && len(args) != initSeWithDCArgCount {
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

func initCE(cmd *cobra.Command, args []string) error {

	cluster := newCluster()
	defer cluster.saveToJSON()

	if !cluster.Draft {
		return ErrorClusterConfAlreadyExists
	}

	err := mkCommandDirAndLogFile(cmd)
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

func initSE(cmd *cobra.Command, args []string) error {

	cluster := newCluster()
	defer cluster.saveToJSON()

	if !cluster.Draft {
		return ErrorClusterConfAlreadyExists
	}

	err := mkCommandDirAndLogFile(cmd)
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
	}

	return err
}
