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
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

var dryRun bool

func saveClusterToJson(c *clusterType) {
	err := c.saveToJSON()
	if err != nil {
		loggerError(err.Error())
	}
}

func newCluster() *clusterType {
	var cluster = clusterType{
		DesiredClusterVersion: version,
		ActualClusterVersion:  "",
		exists:                false,
		Draft:                 true,
		sshKey:                sshKey,
		dryRun:                dryRun,
		SshPort:               sshPort,
		Cmd:                   newCmd("", make([]string, 0)),
		SkipStacks:            make([]string, 0),
		ReplacedAddresses:     make([]string, 0),
		Cron:                  &cronType{},
		Acme:                  &acmeType{Domains: make([]string, 0)},
		Alert:                 &alertType{DiscordWebhook: emptyDiscordWebhookUrl},
	}

	sshKey, exists := os.LookupEnv(envVoedgerSshKey)
	if exists && cluster.sshKey == "" {
		cluster.sshKey = sshKey
	}

	if len(acmeDomains) != 0 {
		cluster.Acme.Domains = strings.Split(acmeDomains, comma)
	}

	if err := cluster.setEnv(); err != nil {
		loggerError(err.Error())
		return nil
	}

	dir, _ := os.Getwd()

	cluster.configFileName = filepath.Join(dir, clusterConfFileName)

	// Preparation of a configuration file for Dry Run mode
	if cluster.dryRun {

		dryRunDir := filepath.Join(dir, dryRunDir)
		exists, err := coreutils.Exists(dryRunDir)
		if err != nil {
			// notest
			loggerError(err.Error())
			return nil
		}
		if !exists {
			err := os.Mkdir(dryRunDir, coreutils.FileMode_rwxrwxrwx)
			if err != nil {
				loggerError(err.Error())
				return nil
			}
		}
		dryRunClusterConfigFileName := filepath.Join(dryRunDir, clusterConfFileName)

		// Remove the old dry run configuration file
		// Under tests, you do not need to delete for the possibility of testing command sequences
		if !testing.Testing() {
			exists, err := coreutils.Exists(dryRunClusterConfigFileName)
			if err != nil {
				// notest
				loggerError(err.Error())
				return nil
			}
			if exists {
				os.Remove(dryRunClusterConfigFileName)
			}
		}

		exists, err = coreutils.Exists(cluster.configFileName)
		if err != nil {
			// notest
			loggerError(err.Error())
			return nil
		}
		if exists {
			if err := coreutils.CopyFile(cluster.configFileName, dryRunClusterConfigFileName); err != nil {
				loggerError(err.Error())
				return nil
			}
		}

		cluster.configFileName = dryRunClusterConfigFileName
	}

	exists, err := cluster.clusterConfigFileExists()
	if err != nil {
		// notest
		loggerError(err.Error())
		return nil
	}
	if exists {
		cluster.exists = true
		if err := cluster.loadFromJSON(); err != nil {
			loggerError(err.Error())
			return nil
		}
	}

	return &cluster
}

func newCmd(cmdKind string, cmdArgs []string) *cmdType {
	return &cmdType{
		Kind: cmdKind,
		Args: cmdArgs,
	}
}

func newNodeState(address string, nodeVersion string) *nodeStateType {
	return &nodeStateType{Address: address, NodeVersion: nodeVersion}
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
	idx              int            // the sequence number of the node, starts with 1
	Error            string         `json:"Error,omitempty"`
	ActualNodeState  *nodeStateType `json:"ActualNodeState,omitempty"`
	DesiredNodeState *nodeStateType `json:"DesiredNodeState,omitempty"`
}

func (n *nodeType) address() string {
	if n.ActualNodeState != nil && len(n.ActualNodeState.Address) > 0 {
		return n.ActualNodeState.Address
	} else if n.DesiredNodeState != nil && len(n.DesiredNodeState.Address) > 0 {
		return n.DesiredNodeState.Address
	}

	err := fmt.Errorf(errEmptyNodeAddress, n.nodeName(), ErrEmptyNodeAddress)
	loggerError(err.Error)
	panic(err)
}

// nolint
func (n *nodeType) nodeName() string {
	if n.cluster.Edition == clusterEditionN5 {
		if n.cluster.SubEdition == clusterSubEditionSE3 {
			return fmt.Sprintf("node-%d", n.idx)
		} else {
			switch n.idx {
			case 1:
				return "app-node-1"
			case 2:
				return "app-node-2"
			case 3:
				return "db-node-1"
			case 4:
				return "db-node-2"
			case 5:
				return "db-node-3"
			default:
				return "node"

			}
		}

	} else if n.cluster.Edition == clusterEditionN1 {
		return n1NodeName
	} else {
		return "node"
	}
}

// nolint
func (n *nodeType) hostNames() []string {
	if n.cluster.SubEdition == clusterSubEditionSE5 {
		return []string{n.nodeName()}
	} else if n.cluster.SubEdition == clusterSubEditionSE3 {
		switch n.idx {
		case 1:
			return []string{"app-node-1", "db-node-1"}
		case 2:
			return []string{"app-node-2", "db-node-2"}
		case 3:
			return []string{"db-node-3"}
		}
	}
	return []string{n.nodeName()}
}

// the minimum amount of RAM required by the node (as string)
// nolint
func (n *nodeType) minAmountOfRAM() string {
	switch n.NodeRole {
	case nrAppNode:
		return minRamOnAppNode
	case nrDBNode:
		return minRamOnDBNode
	default:
		return minRamCENode
	}
}

func (n *nodeType) nodeControllerFunction() error {
	if dryRun {
		if n.DesiredNodeState != nil {
			n.success()
			return nil
		}
	}

	switch n.NodeRole {
	case nrDBNode, nrAppNode, nrAppDbNode:
		return seNodeControllerFunction(n)
	case nrN1Node:
		return ceNodeControllerFunction(n)
	default:
		return ErrNodeControllerFunctionNotAssigned
	}
}

func (n *nodeType) success() {
	if n.DesiredNodeState != nil {
		n.ActualNodeState = newNodeState(n.DesiredNodeState.Address, n.desiredNodeVersion(n.cluster))
		n.DesiredNodeState.clear()
	}
	n.Error = ""
}

// nolint
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
	if n.DesiredNodeState != nil && !n.DesiredNodeState.isEmpty() {
		return n.DesiredNodeState.NodeVersion
	}
	return c.DesiredClusterVersion
}

// nolint
func (n *nodeType) actualNodeVersion() string {
	return n.ActualNodeState.NodeVersion
}

// nolint
func (n *nodeType) label(key string) []string {
	switch n.NodeRole {
	case nrN1Node:
		return []string{"ce"}
	case nrAppNode:
		if key != swarmAppLabelKey {
			return []string{fmt.Sprintf("AppNode%d", n.idx)}
		}
		return []string{"AppNode"}
	case nrDBNode:
		if n.cluster.SubEdition == clusterSubEditionSE3 {
			return []string{fmt.Sprintf("DBNode%d", n.idx)}
		}
		return []string{fmt.Sprintf("DBNode%d", n.idx-seNodeCount)}
	case nrAppDbNode:
		if key != swarmAppLabelKey {
			return []string{fmt.Sprintf("AppNode%d", n.idx), fmt.Sprintf("DBNode%d", n.idx)}
		}
		return []string{"AppNode", fmt.Sprintf("DBNode%d", n.idx)}

	}

	err := fmt.Errorf(errInvalidNodeRole, n.address(), ErrInvalidNodeRole)
	loggerError(err.Error)
	panic(err)
}

// nolint
func (ns *nodeType) check(c *clusterType) error {
	if ns.actualNodeVersion() != ns.desiredNodeVersion(c) {
		return fmt.Errorf(errDifferentNodeVersion, ns.actualNodeVersion(), ns.desiredNodeVersion(c), ErrIncorrectVersion)
	}
	return nil
}

// nolint
type nodesType []*nodeType

// returns a list of node addresses
// you can specify the role of nodes to get addresses
// if role = "", the full list of all cluster nodes will be returned
// nolint
func (n *nodesType) hosts(nodeRole string) []string {
	var h []string
	for _, N := range *n {
		if nodeRole == "" || N.NodeRole == nodeRole {
			h = append(h, N.ActualNodeState.Address)
		}
	}
	return h
}

type cmdArgsType []string

type cmdType struct {
	Kind       string
	Args       cmdArgsType
	SkipStacks []string
}

func (a *cmdArgsType) replace(sourceValue, destValue string) {
	for i, v := range *a {
		if v == sourceValue {
			(*a)[i] = destValue
		}
	}
}

func (c *cmdType) apply(cluster *clusterType) error {

	var err error

	if err = cluster.validate(); err != nil {
		loggerError(err.Error)
		return err
	}

	cluster.Draft = false

	var wg sync.WaitGroup
	wg.Add(len(cluster.Nodes))

	for i := 0; i < len(cluster.Nodes); i++ {
		go func(node *nodeType) {
			defer wg.Done()
			if err := node.nodeControllerFunction(); err != nil {
				loggerError(err.Error)
			}
		}(&cluster.Nodes[i])
	}

	wg.Wait()

	if cluster.existsNodeError() {
		return ErrPreparingClusterNodes
	}

	return cluster.clusterControllerFunction()
}

func (c *cmdType) clear() {
	c.Kind = ""
	c.Args = []string{}
}

func (c *cmdType) isEmpty() bool {

	return c.Kind == "" && len(c.Args) == 0
}

func (c *cmdType) validate(cluster *clusterType) error {
	switch c.Kind {
	case ckInit:
		return validateInitCmd(c, cluster)
	case ckUpgrade:
		return validateUpgradeCmd(c, cluster)
	case ckReplace:
		return validateReplaceCmd(c, cluster)
	case ckBackup:
		return validateBackupCmd(c, cluster)
	case ckAcme:
		return validateAcmeCmd(c, cluster)
	default:
		return ErrUnknownCommand
	}
}

// init [CE] [ipAddr1]
// or
// init [SE] [ipAddr1] [ipAddr2] [ipAddr3] [ipAddr4] [ipAddr5]
// nolint
func validateInitCmd(cmd *cmdType, _ *clusterType) error {

	if len(cmd.Args) == 0 {
		return ErrMissingCommandArguments
	}

	if cmd.Args[0] != clusterEditionN1 && cmd.Args[0] != clusterEditionN5 {
		return ErrInvalidClusterEdition
	}
	if cmd.Args[0] == clusterEditionN1 && len(cmd.Args) != 1+initCeArgCount {
		return ErrInvalidNumberOfArguments
	}

	return nil
}

// update [desiredVersion]
func validateUpgradeCmd(_ *cmdType, _ *clusterType) error {
	return nil
}

func validateReplaceCmd(cmd *cmdType, cluster *clusterType) error {

	if len(cmd.Args) == 0 {
		return ErrMissingCommandArguments
	}

	if len(cmd.Args) != 2 {
		return ErrInvalidNumberOfArguments
	}

	var err error

	if n := cluster.nodeByHost(cmd.Args[0]); n == nil {
		err = errors.Join(err, fmt.Errorf(errHostNotFoundInCluster, cmd.Args[0], ErrHostNotFoundInCluster))
	}

	if n := cluster.nodeByHost(cmd.Args[1]); n != nil {
		err = errors.Join(err, fmt.Errorf(ErrHostAlreadyExistsInCluster.Error(), cmd.Args[1]))
	}

	return err
}

func validateBackupCmd(cmd *cmdType, cluster *clusterType) error {
	if len(cmd.Args) == 0 {
		return ErrMissingCommandArguments
	}

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	if len(cmd.Args) <= 1 {
		return ErrMissingCommandArguments
	}

	switch cmd.Args[0] {
	case "node":
		return validateBackupNodeCmd(cmd, cluster)
	case "cron":
		return validateBackupCronCmd(cmd, cluster)
	default:
		return ErrUnknownCommand
	}
}

func validateAcmeCmd(cmd *cmdType, cluster *clusterType) error {

	if len(cmd.Args) == 0 {
		return ErrMissingCommandArguments
	}

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	switch cmd.Args[0] {
	case "add":
		return validateAcmeAddCmd(cmd, cluster)
	case "remove":
		return validateAcmeRemoveCmd(cmd, cluster)
	default:
		return ErrUnknownCommand
	}

}

func validateAcmeAddCmd(cmd *cmdType, cluster *clusterType) error {

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	if len(cmd.Args) != 2 {
		return ErrInvalidNumberOfArguments
	}

	return nil
}

func validateAcmeRemoveCmd(cmd *cmdType, cluster *clusterType) error {

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	if len(cmd.Args) != 2 {
		return ErrInvalidNumberOfArguments
	}

	domains := strings.Split(cmd.Args[1], comma)
	domainsMap := make(map[string]bool)
	for _, s := range cluster.Acme.Domains {
		domainsMap[s] = true
	}

	var notFound []string
	for _, s := range domains {
		if !domainsMap[s] {
			notFound = append(notFound, s)
		}
	}
	if len(notFound) > 0 {
		return fmt.Errorf(errDomainsNotFound, strings.Join(notFound, comma), ErrDomainsNotFound)
	}

	return nil
}

type cronType struct {
	Backup     string `json:"Backup,omitempty"`
	ExpireTime string `json:"ExpireTime,omitempty"`
}

type acmeType struct {
	Domains []string `json:"Domains,omitempty"`
}

func (a *acmeType) domains() string {
	return strings.Join(a.Domains, comma)
}

// adds new domains to the ACME Domains list from a string "Domain1,Domain2,Domain3"
func (a *acmeType) addDomains(domainsStr string) {
	domains := strings.Split(domainsStr, comma)
	for _, d := range domains {
		if !strings.Contains(strings.Join(a.Domains, comma), d) {
			a.Domains = append(a.Domains, d)
		}
	}
}

// removes domains from the ACME Domains list from a string "Domain1,Domain2,Domain3"
func (a *acmeType) removeDomains(domainsStr string) {
	domains := strings.Split(domainsStr, comma)
	for _, d := range domains {
		for i, v := range a.Domains {
			if v == d {
				a.Domains = append(a.Domains[:i], a.Domains[i+1:]...)
			}
		}
	}
}

type alertType struct {
	DiscordWebhook string `json:"DiscordWebhook,omitempty"`
}

type clusterType struct {
	configFileName        string
	sshKey                string
	exists                bool //the cluster is loaded from "cluster.json" at the start of ctool
	dryRun                bool
	Edition               string
	SubEdition            string `json:"SubEdition,omitempty"`
	ActualClusterVersion  string
	DesiredClusterVersion string     `json:"DesiredClusterVersion,omitempty"`
	SshPort               string     `json:"SSHPort,omitempty"`
	Acme                  *acmeType  `json:"Acme,omitempty"`
	Cmd                   *cmdType   `json:"Cmd,omitempty"`
	LastAttemptError      string     `json:"LastAttemptError,omitempty"`
	SkipStacks            []string   `json:"SkipStacks,omitempty"`
	Cron                  *cronType  `json:"Cron,omitempty"`
	Alert                 *alertType `json:"Alert,omitempty"`
	Nodes                 []nodeType
	ReplacedAddresses     []string `json:"ReplacedAddresses,omitempty"`
	Draft                 bool     `json:"Draft,omitempty"`
}

// map[hostname]ipAddress
func (c *clusterType) hosts() map[string]string {
	hosts := make(map[string]string)
	var addr string
	for _, n := range c.Nodes {
		for i := 0; i < len(n.hostNames()); i++ {
			if n.DesiredNodeState != nil && len(n.DesiredNodeState.Address) > 0 {
				addr = n.DesiredNodeState.Address
			} else {
				addr = n.address()
			}
			hosts[n.hostNames()[i]] = addr
		}
	}
	return hosts
}

// apply the cluster data to the template file
func (c *clusterType) updateTemplateFile(filename string) error {
	return prepareScriptFromTemplate(filename, c)
}

func (c *clusterType) clusterControllerFunction() error {
	if dryRun {
		c.success()
		return nil
	}

	switch c.Edition {
	case clusterEditionN1:
		return ceClusterControllerFunction(c)
	case clusterEditionN5:
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

func (c *clusterType) nodeByHost(addrOrHostName string) *nodeType {

	if c.SubEdition == clusterSubEditionSE3 {
		switch {
		case addrOrHostName == "app-node-1" || addrOrHostName == "db-node-1":
			addrOrHostName = "node-1"
		case addrOrHostName == "app-node-2" || addrOrHostName == "db-node-2":
			addrOrHostName = "node-2"
		case addrOrHostName == "db-node-3":
			addrOrHostName = "node-3"
		}
	}

	for i, n := range c.Nodes {
		if addrOrHostName == n.nodeName() || equalIPs(n.ActualNodeState.Address, addrOrHostName) {
			return &c.Nodes[i]
		}
	}
	return nil
}

func (c *clusterType) applyCmd(cmd *cmdType) error {
	if err := cmd.validate(c); err != nil {
		return err
	}

	if !c.Draft && c != nil && !c.Cmd.isEmpty() {
		return ErrUncompletedCommandFound
	}

	// nolint
	defer saveClusterToJson(c)

	switch cmd.Kind {
	case ckAcme:
		if cmd.Args[0] == "add" && len(cmd.Args) == 2 {
			c.Acme.addDomains(cmd.Args[1])
			if err := c.setEnv(); err != nil {
				return err
			}
		} else if cmd.Args[0] == "remove" && len(cmd.Args) == 2 {
			c.Acme.removeDomains(cmd.Args[1])
			if err := c.setEnv(); err != nil {
				return err
			}
		}
	case ckReplace:
		oldAddr := cmd.Args[0]
		newAddr := cmd.Args[1]
		cmd.SkipStacks = c.SkipStacks

		if c.addressInReplacedList(newAddr) {
			return fmt.Errorf(errAddressInReplacedList, newAddr, ErrAddressCannotBeUsed)
		}

		node := c.nodeByHost(oldAddr)
		if node == nil {
			return fmt.Errorf(errHostNotFoundInCluster, oldAddr, ErrHostNotFoundInCluster)
		}

		if oldAddr == node.nodeName() {
			oldAddr = node.ActualNodeState.Address
			cmd.Args.replace(node.nodeName(), oldAddr)
		}

		if !dryRun {
			if err := nodeIsDown(node); err != nil {
				return fmt.Errorf(errCannotReplaceALiveNode, oldAddr, ErrCommandCannotBeExecuted)
			}

			if err := hostIsAvailable(c, newAddr); err != nil {
				return fmt.Errorf(errHostIsNotAvailable, newAddr, ErrHostIsNotAvailable)
			}

		}

		node.DesiredNodeState = newNodeState(newAddr, node.desiredNodeVersion(c))

		if node.ActualNodeState != nil {
			node.ActualNodeState.clear()
		}
	case ckUpgrade:
		c.DesiredClusterVersion = version
		cmd.SkipStacks = c.SkipStacks
		for i := range c.Nodes {
			c.Nodes[i].DesiredNodeState = newNodeState(c.Nodes[i].ActualNodeState.Address, version)
			/*
				c.Nodes[i].DesiredNodeState.NodeVersion = version
				c.Nodes[i].DesiredNodeState.Address = c.Nodes[i].ActualNodeState.Address
			*/
		}
	}

	c.Cmd = cmd

	return nil
}

func (c *clusterType) updateNodeIndexes() {
	for i := range c.Nodes {
		c.Nodes[i].idx = i + 1
	}
}

// TODO: Filename should be an argument
func (c *clusterType) saveToJSON() error {

	mu.Lock()
	defer mu.Unlock()

	if c.Cmd != nil && c.Cmd.isEmpty() {
		c.Cmd = nil
	}
	for i := 0; i < len(c.Nodes); i++ {
		if c.Nodes[i].DesiredNodeState != nil && c.Nodes[i].DesiredNodeState.isEmpty() {
			c.Nodes[i].DesiredNodeState = nil
		}
		if c.Nodes[i].ActualNodeState != nil && c.Nodes[i].ActualNodeState.isEmpty() {
			c.Nodes[i].ActualNodeState = nil
		}
	}

	b, err := json.Marshal(c)
	if err == nil {
		b, err = prettyprint(b)
		if err != nil {
			return err
		}
		err = os.WriteFile(c.configFileName, b, coreutils.FileMode_rwxrwxrwx)
	}
	return err
}

// The address was replaced in the cluster
func (c *clusterType) addressInReplacedList(address string) bool {
	for _, value := range c.ReplacedAddresses {
		if value == address {
			return true
		}
	}
	return false
}

func (c *clusterType) clusterConfigFileExists() (bool, error) {
	return coreutils.Exists(c.configFileName)
}

func (c *clusterType) loadFromJSON() error {

	defer c.updateNodeIndexes()
	defer func() {
		if c.Cmd == nil {
			c.Cmd = newCmd("", []string{})
		}
		for i := 0; i < len(c.Nodes); i++ {
			if c.Nodes[i].ActualNodeState == nil {
				c.Nodes[i].ActualNodeState = newNodeState("", "")
			}
			if c.Nodes[i].DesiredNodeState == nil {
				c.Nodes[i].DesiredNodeState = newNodeState("", "")
			}
		}
	}()

	exists, err := c.clusterConfigFileExists()
	if err != nil {
		// notest
		return err
	}
	if !exists {
		return ErrClusterConfNotFound
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

	if c.Edition == clusterEditionSE && c.SubEdition != clusterSubEditionSE3 {
		c.SubEdition = clusterSubEditionSE5
	}

	if err == nil {
		err = c.setEnv()
	}

	// Compatibility of versions 0.0.6 (and younger) with version 0.0.7 (and older) (CE, SE3, SE5 -> n1, n3, n5)
	switch c.Edition {
	case clusterEditionCE:
		c.Edition = clusterEditionN1
		c.Nodes[0].NodeRole = nrN1Node
	case clusterEditionSE:
		if c.SubEdition == clusterSubEditionSE3 {
			err = fmt.Errorf("the configuration of the SE3 cluster is not supported. You must use ctool version 0.0.6.")
		} else {
			c.Edition = clusterEditionN5
			c.SubEdition = ""
		}
	}

	return err
}

// Installation of the necessary variables of the environment
func (c *clusterType) setEnv() error {

	setEnv := "Set env %s = %s"

	logger.Verbose(fmt.Sprintf(setEnv, envVoedgerNodeSshPort, c.SshPort))
	if err := os.Setenv(envVoedgerNodeSshPort, c.SshPort); err != nil {
		return err
	}

	logger.Verbose(fmt.Sprintf(setEnv, envVoedgerAcmeDomains, c.Acme.domains()))
	if err := os.Setenv(envVoedgerAcmeDomains, c.Acme.domains()); err != nil {
		return err
	}

	if c.sshKey != "" {
		logger.Verbose(fmt.Sprintf(setEnv, envVoedgerSshKey, c.sshKey))
		if err := os.Setenv(envVoedgerSshKey, c.sshKey); err != nil {
			return err
		}
	}

	if c.Edition == clusterEditionN5 {
		logger.Verbose(fmt.Sprintf(setEnv, envVoedgerEdition, clusterSubEditionSE5))
		if err := os.Setenv(envVoedgerEdition, clusterSubEditionSE5); err != nil {
			return err
		}
	}

	if /*c.Edition == clusterEditionN1 &&*/ len(c.Nodes) == 1 {
		var port string

		if c.Acme != nil && c.Acme.domains() != "" {
			port = httpsPort
		} else {
			port = httpPort
		}

		logger.Verbose(fmt.Sprintf(setEnv, envVoedgerHttpPort, port))
		if err := os.Setenv(envVoedgerHttpPort, port); err != nil {
			return err
		}

		ceNode := c.Nodes[0].address()
		logger.Verbose(fmt.Sprintf(setEnv, envVoedgerCeNode, ceNode))
		if err := os.Setenv(envVoedgerCeNode, ceNode); err != nil {
			return err
		}
	}
	return nil
}

// nolint
func (c *clusterType) readFromInitArgs(cmd *cobra.Command, args []string) error {

	defer c.updateNodeIndexes()

	// nolint
	defer saveClusterToJson(c)

	if cmd == initN1Cmd { // CE args
		c.Edition = clusterEditionN1
		c.Nodes = make([]nodeType, 1)
		c.Nodes[0].NodeRole = nrN1Node
		c.Nodes[0].cluster = c
		c.Nodes[0].DesiredNodeState = newNodeState("", c.DesiredClusterVersion)
		c.Nodes[0].ActualNodeState = newNodeState("", "")
		if len(args) > 0 {
			c.Nodes[0].DesiredNodeState.Address = args[0]
		} else {
			c.Nodes[0].DesiredNodeState.Address = "0.0.0.0"
		}
	} else { // SE args
		skipStacks, err := cmd.Flags().GetStringSlice("skip-stack")
		if err != nil {
			fmt.Println("Error getting skip-stack values:", err)
			return err
		}
		c.SkipStacks = skipStacks

		if len(args) == se5NodeCount {

			c.Edition = clusterEditionN5
			c.SubEdition = ""
			c.Nodes = make([]nodeType, se5NodeCount)

			for i := 0; i < se5NodeCount; i++ {
				if i < seNodeCount {
					c.Nodes[i].NodeRole = nrAppNode
				} else {
					c.Nodes[i].NodeRole = nrDBNode
				}
				c.Nodes[i].DesiredNodeState = newNodeState(args[i], c.DesiredClusterVersion)
				c.Nodes[i].ActualNodeState = newNodeState("", "")
				c.Nodes[i].cluster = c
			}
		} else {
			return ErrInvalidNumberOfArguments
		}

	}
	return c.setEnv()
}

// nolint
func (c *clusterType) validate() error {

	var err error

	for _, n := range c.Nodes {
		if n.DesiredNodeState != nil && len(n.DesiredNodeState.Address) > 0 && net.ParseIP(n.DesiredNodeState.Address) == nil {
			err = errors.Join(err, errors.New(n.DesiredNodeState.Address+" "+ErrInvalidIpAddress.Error()))
		}
		if n.ActualNodeState != nil && len(n.ActualNodeState.Address) > 0 && net.ParseIP(n.ActualNodeState.Address) == nil {
			err = errors.Join(err, errors.New(n.ActualNodeState.Address+" "+ErrInvalidIpAddress.Error()))
		}
	}

	if c.Edition != clusterEditionN1 && c.Edition != clusterEditionN5 {
		err = errors.Join(err, ErrInvalidClusterEdition)
	}

	return err
}

func (c *clusterType) success() {
	c.ActualClusterVersion = c.DesiredClusterVersion
	c.DesiredClusterVersion = ""
	if c.Cmd != nil {
		c.Cmd.clear()
	}
	c.LastAttemptError = ""
}

// nolint
func (c *clusterType) fail(error string) {
	c.LastAttemptError = error
}

// nolint
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

func (c *clusterType) checkVersion() error {

	loggerInfo("Ctool version: ", version)

	var clusterVersion string

	exists, err := c.clusterConfigFileExists()
	if err != nil {
		// notest
		return err
	}
	if exists && !c.Cmd.isEmpty() {
		clusterVersion = c.DesiredClusterVersion
	}

	if len(clusterVersion) == 0 {
		clusterVersion = c.ActualClusterVersion
	}

	// The cluster configuration is still missing
	if clusterVersion == "" {
		loggerInfo("Cluster version is missing")
		return nil
	}

	loggerInfo("Cluster version: ", clusterVersion)

	vr := compareVersions(version, clusterVersion)
	if vr == 1 {
		return fmt.Errorf(errCtoolVersionNewerThanClusterVersion, version, clusterVersion, ErrIncorrectVersion)
	} else if vr == -1 {
		return fmt.Errorf(errClusterVersionNewerThanCtoolVersion, clusterVersion, version, clusterVersion, ErrIncorrectVersion)
	}

	return nil
}

func (c *clusterType) needUpgrade() (bool, error) {
	vr := compareVersions(version, c.ActualClusterVersion)
	if vr == -1 {
		return false, fmt.Errorf(errClusterVersionNewerThanCtoolVersion, c.ActualClusterVersion, version, c.ActualClusterVersion, ErrIncorrectVersion)
	} else if vr == 1 {
		return true, nil
	}

	return false, nil

}
