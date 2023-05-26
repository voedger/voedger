/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

func newCluster() *clusterType {
	var cluster = clusterType{
		DesiredClusterVersion: version,
		ActualClusterVersion:  "",
		exists:                false,
		Draft:                 true,
		sshKey:                sshKey,
	}
	dir, _ := os.Getwd()
	cluster.configFileName = filepath.Join(dir, clusterConfFileName)
	cluster.exists = cluster.loadFromJSON() == nil
	return &cluster
}

func newCmd(cmdKind, cmdArgs string) *cmdType {
	return &cmdType{
		Kind: cmdKind,
		Args: cmdArgs,
	}
}

type nodeStateType struct {
	Address     string `json:"Address,omitempty"`
	NodeVersion string `json:"NodeVersion,omitempty"`
}

func (n *nodeStateType) clear() {
	n.Address = ""
	n.NodeVersion = ""
}

func (n *nodeStateType) isEmpty() bool {
	return n.Address == "" && n.NodeVersion == ""
}

type nodeType struct {
	cluster          *clusterType
	NodeRole         string
	idx              int           // the sequence number of the node, starts with 1
	Error            string        `json:"Error,omitempty"`
	ActualNodeState  nodeStateType `json:"ActualNodeState,omitempty"`
	DesiredNodeState nodeStateType `json:"DesiredNodeState,omitempty"`
}

func (n *nodeType) nodeControllerFunction() error {
	switch n.NodeRole {
	case nrDBNode, nrSENode:
		return seNodeControllerFunction(n)
	case nrCENode:
		return ceNodeControllerFunction(n)
	default:
		return ErrNodeControllerFunctionNotAssigned
	}
}

func (n *nodeType) success() {
	n.ActualNodeState = n.DesiredNodeState
	n.DesiredNodeState.clear()
	n.Error = ""
}

func (n *nodeType) fail(err string) {
	n.Error = err
}

// initializing a new action attempt on a node
// the error is being reset
// the attempt counter is incremented
func (n *nodeType) newAttempt() {
	n.Error = ""
}

func (n *nodeType) desiredNodeVersion(c *clusterType) string {
	if &n.DesiredNodeState != nil && !n.DesiredNodeState.isEmpty() {
		return n.DesiredNodeState.NodeVersion
	}
	return c.DesiredClusterVersion
}

func (n *nodeType) actualNodeVersion() string {
	return n.ActualNodeState.NodeVersion
}

func (n *nodeType) label(key string) string {
	switch n.NodeRole {
	case nrCENode:
		return "ce"
	case nrSENode:
		if key == swarmSeLabelKey {
			return "se"
		} else if key == swarmMonLabelKey {
			return fmt.Sprintf("mon%d", n.idx)
		}
	case nrDBNode:
		return fmt.Sprintf("scylla%d", n.idx-seNodeCount)
	}

	return fmt.Sprintf("node%d", n.idx)
}

func (ns *nodeType) check(c *clusterType) error {
	if ns.actualNodeVersion() != ns.desiredNodeVersion(c) {
		return ErrDifferentNodeVersions
	}
	return nil
}

type nodesType []*nodeType

// returns a list of node addresses
// you can specify the role of nodes to get addresses
// if role = "", the full list of all cluster nodes will be returned
func (n *nodesType) hosts(nodeRole string) []string {
	var h []string
	for _, N := range *n {
		if nodeRole == "" || N.NodeRole == nodeRole {
			h = append(h, N.ActualNodeState.Address)
		}
	}
	return h
}

type cmdType struct {
	Kind string
	Args string
}

func (c *cmdType) apply(cluster *clusterType) error {

	var err error

	defer cluster.saveToJSON()

	if err = cluster.validate(); err != nil {
		logger.Error(err.Error)
		return err
	}

	cluster.Draft = false

	var wg sync.WaitGroup
	wg.Add(len(cluster.Nodes))

	for i := 0; i < len(cluster.Nodes); i++ {
		go func(node *nodeType) {
			defer wg.Done()
			if err := node.nodeControllerFunction(); err != nil {
				logger.Error(err.Error)
			}
		}(&cluster.Nodes[i])
	}

	wg.Wait()

	if cluster.existsNodeError() {
		return ErrPreparingClusterNodes
	}

	return cluster.clusterControllerFunction()
}

func (c *cmdType) args() []string {
	return strings.Split(c.Args, " ")
}

func (c *cmdType) clear() {
	c.Kind = ""
	c.Args = ""
}

func (c *cmdType) isEmpty() bool {

	return c.Kind == "" && c.Args == ""
}

func (c *cmdType) validate(cluster *clusterType) error {
	switch c.Kind {
	case ckInit:
		return validateInitCmd(c, cluster)
	case ckUpgrade:
		return validateUpgradeCmd(c, cluster)
	case ckReplace:
		return validateReplaceCmd(c, cluster)
	default:
		return ErrUnknownCommand
	}
}

// init [CE] [ipAddr1]
// or
// init [SE] [ipAddr1] [ipAddr2] [ipAddr3] [ipAddr4] [ipAddr5]
func validateInitCmd(cmd *cmdType, cluster *clusterType) error {
	args := cmd.args()

	if len(args) == 0 {
		return ErrMissingCommandArguments
	}

	if args[0] != clusterEditionCE && args[0] != clusterEditionSE {
		return ErrInvalidClusterEdition
	}

	if args[0] == clusterEditionCE && len(args) != 1+initCeArgCount ||
		args[0] == clusterEditionSE && len(args) != 1+initSeArgCount && len(args) != initSeWithDCArgCount {
		return ErrInvalidNumberOfArguments
	}

	return nil
}

// update [desiredVersion]
func validateUpgradeCmd(cmd *cmdType, cluster *clusterType) error {
	args := cmd.args()

	if len(args) == 0 {
		return ErrMissingCommandArguments
	}

	if len(args) != 1 {
		return ErrInvalidNumberOfArguments
	}

	if args[0] == cluster.ActualClusterVersion {
		return ErrNoUpdgradeRequired
	}

	return nil
}

// replace [oldIpAddr] [newIpAddr]
func validateReplaceCmd(cmd *cmdType, cluster *clusterType) error {
	args := cmd.args()

	if len(args) == 0 {
		return ErrMissingCommandArguments
	}

	if len(args) != 2 {
		return ErrInvalidNumberOfArguments
	}

	var err error

	if n := cluster.nodeByHost(args[0]); n == nil {
		err = errors.Join(err, fmt.Errorf(ErrHostNotFoundInCluster.Error(), args[0]))
	}

	if n := cluster.nodeByHost(args[1]); n != nil {
		err = errors.Join(err, fmt.Errorf(ErrHostAlreadyExistsInCluster.Error(), args[1]))
	}

	return err
}

type clusterType struct {
	configFileName        string
	sshKey                string
	exists                bool //the cluster is loaded from "cluster.json" at the start of ctool
	Edition               string
	ActualClusterVersion  string
	DesiredClusterVersion string   `json:"DesiredClusterVersion,omitempty"`
	Cmd                   cmdType  `json:"Cmd,omitempty"`
	DataCenters           []string `json:"DataCenters,omitempty"`
	LastAttemptError      string   `json:"LastAttemptError,omitempty"`
	Nodes                 []nodeType
	Draft                 bool   `json:"Draft,omitempty"`
	ManagerToken          string `json:"ManagerToken,omitempty"`
}

func (c *clusterType) clusterControllerFunction() error {
	switch c.Edition {
	case clusterEditionCE:
		return ceClusterControllerFunction(c)
	case clusterEditionSE:
		return seClusterControllerFunction(c)
	default:
		return ErrClusterControllerFunctionNotAssigned
	}
}

func prettyprint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")

	return out.Bytes(), err
}

func equalIPs(ip1, ip2 string) bool {
	netIP1 := net.ParseIP(ip1)
	netIP2 := net.ParseIP(ip2)

	if netIP1 == nil || netIP2 == nil {
		return false
	}

	return netIP1.Equal(netIP2)
}

func (c *clusterType) nodeByHost(address string) *nodeType {
	for i, n := range c.Nodes {
		if equalIPs(n.ActualNodeState.Address, address) {
			return &c.Nodes[i]
		}
	}
	return nil
}

func (c *clusterType) applyCmd(cmd *cmdType) error {
	if err := cmd.validate(c); err != nil {
		return err
	}

	if !c.Draft && !c.Cmd.isEmpty() {
		return ErrUncompletedCommandFound
	}

	c.Cmd = *cmd

	defer c.saveToJSON()
	switch cmd.Kind {
	case ckReplace:
		oldAddr := cmd.args()[0]
		newAddr := cmd.args()[1]
		node := c.nodeByHost(oldAddr)
		node.DesiredNodeState = node.ActualNodeState
		node.DesiredNodeState.Address = newAddr
		node.ActualNodeState.clear()

	}

	return nil
}

func (c *clusterType) updateNodeIndexes() {
	for i := range c.Nodes {
		c.Nodes[i].idx = i + 1
	}
}

func (c *clusterType) saveToJSON() error {

	b, err := json.Marshal(c)
	if err == nil {
		b, err = prettyprint(b)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(c.configFileName, b, rwxrwxrwx)
	}
	return err
}

func (c *clusterType) loadFromJSON() error {

	defer c.updateNodeIndexes()
	if _, err := os.Stat(c.configFileName); err != nil {
		return err
	}

	b, err := os.ReadFile(c.configFileName)
	if err == nil {
		oldDraft := c.Draft
		c.Draft = false
		err = json.Unmarshal(b, c)
		if err != nil {
			c.Draft = oldDraft
		}
	}

	for i := 0; i < len(c.Nodes); i++ {
		c.Nodes[i].cluster = c
	}

	return err
}

func (c *clusterType) readFromInitArgs(cmd *cobra.Command, args []string) error {

	defer c.updateNodeIndexes()

	if cmd == initCECmd { // CE args
		c.Edition = clusterEditionCE
		c.Nodes = make([]nodeType, 1)
		c.Nodes[0].NodeRole = nrCENode
		c.Nodes[0].cluster = c
		c.Nodes[0].DesiredNodeState.NodeVersion = c.DesiredClusterVersion
		if len(args) > 0 {
			c.Nodes[0].DesiredNodeState.Address = args[0]
		} else {
			c.Nodes[0].DesiredNodeState.Address = "0.0.0.0"
		}
	} else { // SE args
		c.Edition = clusterEditionSE
		c.Nodes = make([]nodeType, 5)
		c.DataCenters = make([]string, 0)

		for i := 0; i < initSeArgCount; i++ {
			if i < seNodeCount {
				c.Nodes[i].NodeRole = nrSENode
			} else {
				c.Nodes[i].NodeRole = nrDBNode
			}
			c.Nodes[i].DesiredNodeState.Address = args[i]
			c.Nodes[i].DesiredNodeState.NodeVersion = c.DesiredClusterVersion
			c.Nodes[i].cluster = c
		}

		if len(args) == initSeWithDCArgCount {
			c.DataCenters = append(c.DataCenters, args[seNodeCount:]...)
		}
	}
	return nil
}

func (c *clusterType) validate() error {

	var err error

	for _, n := range c.Nodes {
		if len(n.DesiredNodeState.Address) > 0 && net.ParseIP(n.DesiredNodeState.Address) == nil {
			err = errors.Join(err, errors.New(n.DesiredNodeState.Address+" "+ErrInvalidIpAddress.Error()))
		}
		if len(n.ActualNodeState.Address) > 0 && net.ParseIP(n.ActualNodeState.Address) == nil {
			err = errors.Join(err, errors.New(n.ActualNodeState.Address+" "+ErrInvalidIpAddress.Error()))
		}
	}

	if c.Edition != clusterEditionCE && c.Edition != clusterEditionSE {
		err = errors.Join(err, ErrInvalidClusterEdition)
	}

	if len(c.DataCenters) > 0 && len(c.DataCenters) != 3 {
		err = errors.Join(err, ErrInvalidNumberOfDataCenters)
	}

	return err
}

func (c *clusterType) success() {
	c.ActualClusterVersion = c.DesiredClusterVersion
	c.DesiredClusterVersion = ""
	c.Cmd.clear()
	c.LastAttemptError = ""
}

func (c *clusterType) fail(error string) {
	c.LastAttemptError = error
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := user.Current()
		if err != nil {
			return "", err
		}

		path = filepath.Join(homeDir.HomeDir, path[2:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

func (c *clusterType) existsNodeError() bool {
	for _, n := range c.Nodes {
		if len(n.Error) > 0 {
			return true
		}
	}
	return false
}
