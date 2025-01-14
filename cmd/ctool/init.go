/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"net"

	"github.com/spf13/cobra"
)

var (
	initCECmd, initSECmd *cobra.Command
)
var skipStacks []string

func newInitCmd() *cobra.Command {
	initCECmd = &cobra.Command{
		Use:   "CE [<ipaddr>...]",
		Short: "Create the cluster.json file for the CE edition cluster",
		Args:  cobra.MaximumNArgs(1),
		RunE:  initCE,
	}

	initSECmd = &cobra.Command{
		Use:   "SE [<ipaddr>...]",
		Short: "Create the cluster.json file for the SE edition cluster",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != se3NodeCount && len(args) != se5NodeCount {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: initSE,
	}

	if !addSshKeyFlag(initSECmd) {
		return nil
	}

	initSECmd.Flags().StringSliceVar(&skipStacks, "skip-stack", []string{}, "Specify docker compose stacks to skip")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Creates the file cluster.json for cluster",
	}

	initCmd.PersistentFlags().StringVar(&acmeDomains, "acme-domain", "", "ACME domains <comma separated list>")

	initCmd.PersistentFlags().StringVarP(&sshPort, "ssh-port", "p", "22", "SSH port")

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

	currentCmd = cmd
	cluster := newCluster()
	var err error

	defer saveClusterToJson(cluster)

	if !cluster.Draft {
		return ErrClusterConfAlreadyExists
	}

	c := newCmd(ckInit, append([]string{"CE"}, args...))
	if err = cluster.applyCmd(c); err != nil {
		loggerError(err.Error())
		return err
	}

	err = mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	err = cluster.readFromInitArgs(cmd, args)
	if err != nil {
		loggerError(err.Error())
		return err
	}

	if err = cluster.Cmd.apply(cluster); err != nil {
		loggerError(err)
		return err
	}

	return err
}

// nolint
func initSE(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
	cluster := newCluster()
	var err error
	if !cluster.Draft {
		return ErrClusterConfAlreadyExists
	}

	c := newCmd(ckInit, append([]string{"SE"}, args...))
	c.SkipStacks = skipStacks
	if err = cluster.applyCmd(c); err != nil {
		loggerError(err.Error())
		return err
	}

	defer saveClusterToJson(cluster)

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
			loggerError(err)
			return err
		}
	}

	return nil
}
