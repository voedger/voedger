/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"embed"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/voedger/voedger/pkg/goutils/exec"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"golang.org/x/term"
)

//go:embed scripts/drafts/*
var scriptsFS embed.FS

var scriptsTempDir string

var indicator []string

type scriptExecuterType struct {
	outputPrefix string
	sshKeyPath   string
}

func selectIndicator() []string {
	indicators1 := []string{"|", "/", "-", "\\"}
	indicators2 := []string{"◐", "◓", "◑", "◒"}
	indicators3 := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

	indicators := [][]string{indicators1, indicators2, indicators3}
	// nolint
	randomIndex := rand.Intn(len(indicators))
	return indicators[randomIndex]
}

func showProgress(done chan bool) {

	if len(indicator) == 0 {
		indicator = selectIndicator()
	}

	i := 0
	for {
		select {
		case <-done:
			fmt.Print("\r")
			return
		default:
			if !verbose() {
				fmt.Printf(green("\r%s\r"), indicator[i])
			}
			i = (i + 1) % len(indicator)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func verbose() bool {
	if dryRun {
		return true
	}
	b, err := rootCmd.Flags().GetBool("verbose")
	return err == nil && b
}

func (se *scriptExecuterType) run(scriptName string, args ...string) error {

	var pExec *exec.PipedExec

	// nolint
	os.Chdir(scriptsTempDir)

	args = append([]string{scriptName}, args...)
	pExec = new(exec.PipedExec).Command("bash", args...)

	var stdoutWriter io.Writer
	var stderrWriter io.Writer
	if logFile != nil {
		if verbose() {
			stdoutWriter = io.MultiWriter(os.Stdout, logFile)
			stderrWriter = io.MultiWriter(os.Stderr, logFile)
		} else {
			stdoutWriter = logFile
			stderrWriter = logFile
		}
	} else {
		if verbose() {
			stdoutWriter = os.Stdout
			stderrWriter = os.Stderr
		} else {
			stdoutWriter = nil
			stderrWriter = nil
		}
	}

	done := make(chan bool)
	go showProgress(done)
	defer func() { done <- true }()

	var err error
	if len(se.outputPrefix) > 0 {
		sedArg := fmt.Sprintf("s/^/[%s]: /", se.outputPrefix)
		err = pExec.
			Command("sed", sedArg).
			Run(stdoutWriter, stderrWriter)
	} else {
		err = pExec.
			Run(stdoutWriter, stderrWriter)
	}

	if err != nil && verbose() {
		loggerError(fmt.Errorf("the error of the script %s: %w", scriptName, err).Error())
	}
	return err
}

func newScriptExecuter(sshKey string, outputPrefix string) *scriptExecuterType {
	return &scriptExecuterType{sshKeyPath: sshKey, outputPrefix: outputPrefix}
}

// nolint
func getEnvValue1(key string) string {
	value, _ := os.LookupEnv(key)
	return value
}

func prepareScripts(scriptFileNames ...string) error {

	// nolint
	os.Chdir(scriptsTempDir)

	err := createScriptsTempDir()
	if err != nil {
		return err
	}

	// If scriptfilenames is empty, then we will copy all scripts from scriptsfs
	err = coreutils.CopyDirFS(scriptsFS, "scripts/drafts", scriptsTempDir, coreutils.WithFilterFilesWithRelativePaths(scriptFileNames),
		coreutils.WithSkipExisting())
	if err != nil {
		loggerError(err.Error())
		return err
	}
	return nil
}

// nolint
func inputPassword(pass *string) error {

	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err == nil {
		*pass = string(bytePassword)
		return nil
	}
	return err
}

// nolint
func prepareScriptFromTemplate(scriptFileName string, data interface{}) error {

	err := createScriptsTempDir()
	if err != nil {
		return err
	}

	tmpl, err := template.ParseFS(scriptsFS, filepath.Join(embedScriptsDir, scriptFileName))
	if err != nil {
		return err
	}

	destFilename := filepath.Join(scriptsTempDir, scriptFileName)
	destFile, err := os.Create(destFilename)
	if err != nil {
		return err
	}
	defer destFile.Close()

	err = destFile.Chmod(coreutils.FileMode_rw_rw_rw_)
	if err != nil {
		return err
	}

	err = tmpl.Execute(destFile, data)
	if err != nil {
		return err
	}

	return nil
}
