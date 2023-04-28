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

	"github.com/spf13/cobra"
)

func newCluster() *clusterType {
	var cluster = clusterType{
		CToolVersion: version,
		exists:       false,
		Draft:        true,
	}
	dir, _ := os.Getwd()
	cluster.configFileName = filepath.Join(dir, clusterConfFileName)
	cluster.exists = cluster.loadFromJSON() == nil
	return &cluster
}

type nodeStateType struct {
	Address     string
	NodeVersion string
	AttemptNo   int
	Info        string `json:"Info,omitempty"`
	Error       string `json:"Error,omitempty"`
}

// initializing a new action attempt on a node
// the error is being reset
// the attempt counter is incremented
func (ns *nodeStateType) newAttempt() {
	ns.AttemptNo += 1
	ns.Error = ""
}

func (ns *nodeStateType) reset() {
	ns.AttemptNo = 0
	ns.Error = ""
}

type nodeType struct {
	NodeRole string
	idx      int // the sequence number of the node, starts with 1
	State    nodeStateType
}

// label for swarm node
func (n *nodeType) label() string {
	switch n.NodeRole {
	case nrCENode:
		return "ceapp"
	case nrSENode:
		return fmt.Sprintf("app%d", n.idx)
	case nrDBNode:
		return fmt.Sprintf("scylla%d", n.idx-seNodeCount)
	default:
		return fmt.Sprintf("node%d", n.idx)
	}
}

func (ns *nodeType) check(c *clusterType) error {
	if ns.State.NodeVersion != c.CToolVersion {
		return ErrorDifferentNodeVersions
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
			h = append(h, N.State.Address)
		}
	}
	return h
}

type clusterType struct {
	configFileName   string
	sshKey           string
	exists           bool //the cluster is loaded from "cluster.json" at the start of ctool
	Edition          string
	CToolVersion     string
	DataCenters      []string `json:"DataCenters,omitempty"`
	LastAttemptError string   `json:"LastAttemptError,omitempty"`
	LastAttemptInfo  string   `json:"LastAttemptInfo,omitempty"`
	Nodes            []nodeType
	Draft            bool `json:"Draft,omitempty"`
}

func prettyprint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")

	return out.Bytes(), err
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

func (c *clusterType) nodesForProcess() (nodesType, error) {
	nodes := nodesType{}
	var err error
	for i := 0; i < len(c.Nodes); i++ {
		if c.Nodes[i].check(c) == nil {
			continue
		}

		if c.Nodes[i].State.Error == "" {
			nodes = append(nodes, &c.Nodes[i])
			continue
		}

		err = errors.Join(err, errors.New(c.Nodes[i].State.Error))
	}

	return nodes, err
}

func (c *clusterType) needStartProcess() bool {
	exists := false
	for i := range c.Nodes {
		e := c.Nodes[i].check(c)
		if e != nil {
			exists = true
			c.Nodes[i].State.newAttempt()
		}
	}
	return exists
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
	return err
}

func (c *clusterType) readFromInitArgs(cmd *cobra.Command, args []string) error {

	defer c.updateNodeIndexes()

	if cmd == initCECmd { // CE args
		c.Edition = clusterEditionCE
		c.Nodes = make([]nodeType, 1)
		c.Nodes[0].NodeRole = nrCENode
		if len(args) > 0 {
			c.Nodes[0].State.Address = args[0]
		} else {
			c.Nodes[0].State.Address = "0.0.0.0"
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
			c.Nodes[i].State.Address = args[i]
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
		if net.ParseIP(n.State.Address) == nil {
			err = errors.Join(err, errors.New(n.State.Address+" "+ErrorInvalidIpAddress.Error()))
		}
	}

	if c.Edition != clusterEditionCE && c.Edition != clusterEditionSE {
		err = errors.Join(err, ErrorInvalidClusterEdition)
	}

	if len(c.DataCenters) > 0 && len(c.DataCenters) != 3 {
		err = errors.Join(err, ErrorInvalidNumberOfDataCenters)
	}

	return err
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
