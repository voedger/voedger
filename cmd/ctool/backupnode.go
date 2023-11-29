/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/crypto/ssh"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Rman struct {
	Node *Node
}

func (n *Node) Backup(ctx context.Context) (result *DBNodeResult) {
	var (
		err         error
		containerID string
		snapshot    []byte
	)

	if containerID, err = n.nodeTool.init(ctx); err != nil {
		logger.Error(fmt.Errorf("error init node: %w", err))
		return n.NodeError(err)
	}

	if r := n.nodeTool.Run(ctx, exec.Command("touch", "$HOME/ctool/.voedgerbackup")); r.Error != nil {
		logger.Error(fmt.Errorf("error set signal file: %w", r.Error))
		return n.NodeError(r.Error)
	}

	defer func(ctx context.Context, cmd *exec.Cmd) {
		r := n.nodeTool.Run(ctx, cmd)
		if r.Error != nil {
			logger.Error(fmt.Errorf("error cleanup signal file %w", r.Error))
		}
	}(ctx, exec.Command("rm", "$HOME/ctool/.voedgerbackup"))

	if snapshot, err = n.nodeTool.takeSnapshot(ctx); err != nil {
		return n.NodeError(err)
	}
	n.snapShotLabel = string(snapshot)

	defer func(nodeTool *NodeTool, ctx context.Context, snapshotName string) {
		_, err = nodeTool.clearSnapshot(ctx, snapshotName)
		if err != nil {
			logger.Error(fmt.Errorf("error: %w", err))
		}
	}(n.nodeTool, ctx, n.snapShotLabel)

	// Upload backup from snapshot to back up folder
	_, err = n.nodeTool.upload(ctx)
	if err != nil {
		logger.Error(fmt.Errorf("upload snapshot error: %w", err))
		return n.NodeError(err)
	}

	// Add keyspace schema to back up folder
	_, err = n.nodeTool.dumpSchema(ctx)
	if err != nil {
		logger.Error(fmt.Errorf("dump schema error: %w", err))
		return n.NodeError(err)
	}

	return &DBNodeResult{Node: n.Host, Out: containerID}
}

func (n *Node) NodeError(err error) *DBNodeResult {
	return &DBNodeResult{Error: err, Node: n.Host}
}

func (r *Rman) Run(ctx context.Context, callback DBNodeCallback) *DBNodeResult {
	return callback(ctx)
}

type DBNodeCallback func(ctx context.Context) *DBNodeResult

type DBNodeResult struct {
	Node  string
	Out   interface{}
	Error error
}

type DBClusterResult struct {
	Report map[string]*DBNodeResult
}

type Node struct {
	Host          string
	ContainerID   string
	keyspaces     []string
	DataDir       string
	BackupDir     string
	SshCmd        *SshCmdRunner
	nodeTool      *NodeTool
	snapShotLabel string
}

func NewNode(host string, backupDir string, sshCmd *SshCmdRunner) *Node {
	var scyllaHome = "/var/lib/scylla"
	node := &Node{
		Host:      host,
		DataDir:   scyllaHome,
		SshCmd:    sshCmd,
		BackupDir: backupDir,
	}
	node.nodeTool = NewNodeTool("cassandra", "cassandra", node)
	return node
}

type SshCmdRunner struct {
	options *SshOptions
	client  *ssh.Client
}

type SshOptions struct {
	Host       string
	Port       int
	Username   string
	PrivateKey string
}

func NewSshOptions(host string, privateKey string) (*SshOptions, error) {
	var (
		port int
		err  error
	)
	username := os.Getenv("LOGNAME")
	if username == "" {
		return nil, errors.New("LOGNAME environment variable is not set")
	}
	envVarValueSshPort := os.Getenv("VOEDGER_NODE_SSH_PORT")
	if envVarValueSshPort == "" {
		port = 22
	} else {
		port, err = strconv.Atoi(envVarValueSshPort)
		if err != nil {
			return nil, fmt.Errorf("failed to convert SSH port to integer: %w", err)
		}
	}
	return &SshOptions{
		Host:       host,
		Port:       port,
		Username:   username,
		PrivateKey: privateKey,
	}, nil
}

func NewSshCmdRunner(options *SshOptions) (*SshCmdRunner, error) {
	privateKey, err := os.ReadFile(options.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: options.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", options.Host, options.Port), config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH server: %w", err)
	}

	fmt.Print("Connected to SSH server\n" + options.Host + "\n")

	return &SshCmdRunner{
		options: options,
		client:  client,
	}, nil

}

// RunCommand runs a nodetool command with a given context and timeout.
func (e *SshCmdRunner) RunCommand(ctx context.Context, nodetoolCmd *exec.Cmd) ([]byte, error) {
	var session *ssh.Session
	var err error
	// Create a new session for each nodetool command
	if session, err = e.client.NewSession(); err != nil {
		logger.Error(fmt.Errorf("failed to create session: %w", err))
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	logger.Verbose("Run command: " + nodetoolCmd.String() + "\n")

	// Channel to communicate errors from the goroutine
	errChan := make(chan error, 1)

	// Channel to communicate the output from the goroutine
	outputChan := make(chan []byte)

	// Run the nodetool command in a goroutine
	go func() {
		// Ensure the session is closed when the goroutine completes
		defer func(session *ssh.Session) {
			err := session.Close()
			if err != nil {
				errChan <- fmt.Errorf("failed to close session: %w", err)
			}
		}(session)

		// Run the nodetool command on the remote server
		// Run the nodetool command on dethe remote server
		output, err := session.CombinedOutput(nodetoolCmd.String())

		if err != nil {
			errChan <- fmt.Errorf("failed to run nodetool command: %w. %s", err, output)
		} else {
			outputChan <- output
		}
	}()

	defer close(errChan)
	defer close(outputChan)

	select {
	case <-ctx.Done():
		// Wait for either the context to be done or the command to complete
		err := session.Close()
		if err != nil {
			logger.Error(fmt.Errorf("failed to close session: %w", err))
		}
		return nil, ctx.Err()
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	case output := <-outputChan:
		// nodetool command completed successfully
		// Return the output
		return output, nil
	}
	return nil, nil
}

// Close closes the SSH connection.
func (e *SshCmdRunner) Close() {
	if e.client != nil {
		err := e.client.Close()
		if err != nil {
			return
		}
	}
}

type NodeTool struct {
	username string
	password string
	node     *Node
}

func NewRman(nodeName string, targetFolder string, sshKey string) (*Rman, error) {
	var (
		options   *SshOptions
		sshRunner *SshCmdRunner
		err       error
	)
	if options, err = NewSshOptions(nodeName, sshKey); err != nil {
		return nil, err
	}
	if sshRunner, err = NewSshCmdRunner(options); err != nil {
		return nil, err
	}
	return &Rman{
		Node: NewNode(nodeName, targetFolder, sshRunner),
	}, nil
}

func NewNodeTool(username string, password string, node *Node) *NodeTool {
	return &NodeTool{
		username: username,
		password: password,
		node:     node,
	}
}

func (n *NodeTool) init(ctx context.Context) (containerID string, err error) {
	var (
		keyspaces []string
	)
	// Get the ScyllaDB container ID
	containerID, err = n.getContainerID(ctx)
	if err != nil {
		return "", err
	}
	n.node.ContainerID = containerID
	if keyspaces, err = n.descKeyspaces(ctx); err != nil {
		return "", err
	}
	if keyspaces == nil {
		return "", errors.New("no keyspaces found")
	}
	n.node.keyspaces = keyspaces

	logger.Info("Init back up for keyspaces:")

	for _, ks := range n.node.keyspaces {
		logger.Info(ks)
	}
	return containerID, nil
}

func (n *NodeTool) dumpSchema(ctx context.Context) (output []byte, err error) {
	for _, ks := range n.node.keyspaces {
		cmd := exec.Command("docker",
			"exec", n.node.ContainerID,
			fmt.Sprintf("cqlsh -e 'DESC KEYSPACE %s'", ks),
			"|",
			"sed", "-e", "'/^Warning: cqlshrc/d; /^$/d'",
			">",
			fmt.Sprintf(n.node.BackupDir+"/%s/schema.cql", ks),
		)
		output, err = n.node.SshCmd.RunCommand(ctx, cmd)
		if err != nil {
			return output, err
		}
		logger.Info(fmt.Sprintf("Dump schema for keyspace %s", ks))
	}
	return output, nil
}

func cleanupCQL(output []byte) []byte {
	pattern := `(?m)^Warning: cqlshrc.*$\n|^$\n`
	regexpPattern := regexp.MustCompile(pattern)
	return regexpPattern.ReplaceAll(output, []byte{})
}
func (n *NodeTool) descKeyspaces(ctx context.Context) ([]string, error) {
	cmd := exec.Command("docker",
		"exec", n.node.ContainerID,
		"cqlsh", "-e", "'DESC KEYSPACES'",
	)
	output, err := n.node.SshCmd.RunCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return parseKeyspaces(cleanupCQL(output)), nil
}

func snapshotName() string {
	currentTime := time.Now().UTC()
	timeFormat := "2006-01-02_15-04-05"
	formattedTime := currentTime.Format(timeFormat)
	return formattedTime
}

func (n *NodeTool) takeSnapshot(ctx context.Context) (output []byte, err error) {
	snapShotLabel := snapshotName()
	cmd := exec.Command("docker", "exec", n.node.ContainerID,
		"nodetool", "snapshot", "-t",
		snapShotLabel)
	output, err = n.node.SshCmd.RunCommand(ctx, cmd)
	if err != nil {
		return output, err
	}
	return []byte(snapShotLabel), nil
}

func (n *NodeTool) clearSnapshot(ctx context.Context, snapshotName string) (output []byte, err error) {
	cmd := exec.Command("docker", "exec", n.node.ContainerID,
		"nodetool",
		"clearsnapshot",
		"-t",
		snapshotName)
	output, err = n.node.SshCmd.RunCommand(ctx, cmd)
	if err != nil {
		return output, err
	}
	return output, nil
}

func (n *NodeTool) Run(ctx context.Context, cmd *exec.Cmd) *DBNodeResult {
	output, err := n.node.SshCmd.RunCommand(ctx, cmd)
	return &DBNodeResult{
		Node:  n.node.Host,
		Out:   output,
		Error: err,
	}
}

func (n *NodeTool) folderPrepare(ks string) (cmd *exec.Cmd) {
	cmd = exec.Command("mkdir", "-p", fmt.Sprintf("%s/%s", n.node.BackupDir, ks))
	return cmd
}

func (n *NodeTool) upload(ctx context.Context) (output []byte, err error) {
	for _, ks := range n.node.keyspaces {
		n.Run(ctx, n.folderPrepare(ks))
		cmd := exec.Command("sh", "-c",
			fmt.Sprintf("'cd %s/data && find . -type d -print0 | grep -z -iE '/%s/[^/]+/snapshots/%s' | tar -cvzf %s/%s/data.tar.gz --null -T -'",
				n.node.DataDir, ks, n.node.snapShotLabel, n.node.BackupDir, ks),
		)
		output, err = n.node.SshCmd.RunCommand(ctx, cmd)
		if err != nil {
			logger.Error(fmt.Errorf("error upload snapshot: %w", err))
			return output, err
		}
		logger.Info(fmt.Sprintf("Upload snapshot for keyspace %s", ks))
	}
	return output, nil
}

func (n *NodeTool) getContainerID(ctx context.Context) (string, error) {
	containerIDCmd := exec.Command("docker", "ps", "-q", "-f", "name=scylla")
	containerID, err := n.node.SshCmd.RunCommand(ctx, containerIDCmd)
	if err != nil {
		logger.Error(fmt.Errorf("error get container ID: %w", err))
		return "", err
	}
	return strings.TrimSpace(strings.TrimSuffix(string(containerID), "\n")), nil
}

func parseKeyspaces(output []byte) []string {
	lines := splitLines(string(output))
	var keyspaces []string
	for _, line := range lines {
		// Add your logic to filter and extract keyspaces
		if line != "" {
			keyspaces = append(keyspaces, line)
		}
	}
	return keyspaces
}
func splitLines(line string) []string {
	// Split the line into words
	words := strings.Fields(line)

	// Trim each word
	var trimmedWords []string
	for _, word := range words {
		trimmedWords = append(trimmedWords, strings.TrimSpace(word))
	}

	return trimmedWords
}
