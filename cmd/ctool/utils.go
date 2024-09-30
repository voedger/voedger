/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

var logFile *os.File
var commandDirName string

var mutex = &sync.Mutex{}

func loggerInfo(args ...interface{}) {
	if !verbose() {
		fmt.Println(args...)
	}
	logger.Info(args...)
}

func loggerInfoGreen(args ...interface{}) {
	if !verbose() {
		fmt.Println(green(args...))
	}
	logger.Info(args...)
}

func formatArgs(args []interface{}) string {
	formattedArgs := make([]string, len(args))
	for i, arg := range args {
		formattedArgs[i] = fmt.Sprint(arg)
	}
	return strings.Join(formattedArgs, " ")
}

func loggerError(args ...interface{}) {
	if !verbose() {
		s := fmt.Sprintf("%s %s", red("Error:"), formatArgs(args))
		fmt.Println(s)
	}
	logger.Error(args...)
}

func printLogLine(logLevel logger.TLogLevel, line string) {
	line = "\r" + line
	if logFile != nil {
		mutex.Lock()
		fmt.Fprintln(logFile, line)
		mutex.Unlock()
	}
	if logLevel == 1 {
		line = red(line)
	}
	if verbose() {
		logger.DefaultPrintLine(logLevel, line)
	}
}

func getLoggerLevel() logger.TLogLevel {
	if trace() {
		return logger.LogLevelTrace
	}
	if verbose() {
		return logger.LogLevelVerbose
	}
	return logger.LogLevelInfo
}

func mkCommandDirAndLogFile(cmd *cobra.Command, cluster *clusterType) error {
	var (
		s     string
		parts []string
	)

	for cmd.Parent() != nil {
		parts = strings.Split(cmd.Use, " ")
		if s == "" {
			s = parts[0]
		} else {
			s = fmt.Sprintf("%s-%s", parts[0], s)
		}
		cmd = cmd.Parent()
	}

	if cluster.Cmd != nil && !cluster.Cmd.isEmpty() && !strings.Contains(s, cluster.Cmd.Kind) {
		s = fmt.Sprintf("%s-%s", s, cluster.Cmd.Kind)
	}

	time.Sleep(time.Second * 1)
	commandDirName = filepath.Join(logFolder, fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), s))

	if cluster.dryRun {
		commandDirName = filepath.Join(dryRunDir, commandDirName)
	}

	err := os.MkdirAll(commandDirName, coreutils.FileMode_rwxrwxrwx)
	if err == nil {
		fName := filepath.Join(commandDirName, s+".log")
		logFile, err = os.Create(fName)
		if err == nil {
			logFile, err = os.OpenFile(fName, os.O_RDWR, coreutils.FileMode_rw_rw_rw_)
		}
	}
	return err
}

// creates a temporary folder for running scripts, if it doesn't exist
func createScriptsTempDir() error {
	exists, err := scriptTempDirExists()
	if err != nil {
		// notest
		return err
	}
	if exists {
		return nil
	}
	var dir string
	if dir, err = os.MkdirTemp("", "scripts"); err != nil {
		return err
	}
	scriptsTempDir = dir
	if err = os.Chmod(scriptsTempDir, coreutils.FileMode_rwxrwxrwx); err != nil {
		return err
	}

	return nil
}

func scriptTempDirExists() (bool, error) {
	return coreutils.Exists(scriptsTempDir)
}

// deletes the temporary scripts folder, if it exists
func deleteScriptsTempDir() error {
	exists, err := scriptTempDirExists()
	if err != nil {
		// notest
		return err
	}
	if !exists {
		return nil
	}
	return os.RemoveAll(scriptsTempDir)
}

// nolint
func captureStdoutStderr(f func() error) (stdout string, stderr string, err error) {

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		return
	}

	{
		origStdout := os.Stdout
		os.Stdout = stdoutWriter
		defer func() { os.Stdout = origStdout }()
	}
	{
		origStderr := os.Stderr
		os.Stderr = stderrWriter
		defer func() { os.Stderr = origStderr }()
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		var b bytes.Buffer
		defer wg.Done()
		_, _ = io.Copy(&b, stdoutReader)
		stdout = b.String()
	}()
	wg.Add(1)
	go func() {
		var b bytes.Buffer
		defer wg.Done()
		_, _ = io.Copy(&b, stderrReader)
		stderr = b.String()
	}()

	err = f()
	stderrWriter.Close()
	stdoutWriter.Close()
	wg.Wait()
	return

}

// nolint
func randomPassword(length int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	passwordBytes := make([]byte, length)
	for i := range passwordBytes {
		// nolint
		passwordBytes[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(passwordBytes)
}
