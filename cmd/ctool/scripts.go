/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/untillpro/goutils/exec"
	"golang.org/x/crypto/ssh/terminal"
)

//go:embed scripts/drafts/*
var scriptsFS embed.FS

var scriptsTempDir string

type scriptExecuterType struct {
	outputPrefix string
	sshKeyPath   string
}

func showProgress(done chan bool) {
	indicators := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		select {
		case <-done:
			fmt.Print("\r")
			return
		default:
			if !verbose() {
				fmt.Printf(green("\r%s\r"), indicators[i])
			}
			i = (i + 1) % len(indicators)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func verbose() bool {
	b, err := rootCmd.Flags().GetBool("verbose")
	return err == nil && b
}

func (se *scriptExecuterType) run(scriptName string, args ...string) error {

	var pExec *exec.PipedExec

	if len(se.sshKeyPath) > 0 {
		args = append([]string{fmt.Sprintf("eval $(ssh-agent -s); ssh-add %s; ./%s", se.sshKeyPath, scriptName)}, args...)
		pExec = new(exec.PipedExec).Command("bash", "-c", strings.Join(args, " "))
	} else {
		args = append([]string{scriptName}, args...)
		pExec = new(exec.PipedExec).Command("bash", args...)
	}

	os.Chdir(scriptsTempDir)

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
		stdoutWriter = os.Stdout
		stderrWriter = os.Stderr
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

	return err
}

func newScriptExecuter(sshKey string, outputPrefix string) *scriptExecuterType {
	return &scriptExecuterType{sshKeyPath: sshKey, outputPrefix: outputPrefix}
}

func getEnvValue1(key string) string {
	value, _ := os.LookupEnv(key)
	return value
}

func scriptExists(scriptFileName string) bool {
	if scriptsTempDir == "" {
		return false
	}

	if _, err := os.Stat(filepath.Join(scriptsTempDir, scriptFileName)); err == nil {
		return true
	}

	return false
}

func prepareScripts(scriptFileNames ...string) error {

	var m sync.Mutex
	m.Lock()
	defer m.Unlock()

	os.Chdir(scriptsTempDir)

	err := createScriptsTempDir()
	if err != nil {
		return err
	}

	for _, fileName := range scriptFileNames {

		if scriptExists(fileName) {
			continue
		}

		file, err := scriptsFS.Open("scripts/drafts/" + fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		destFileName := filepath.Join(scriptsTempDir, fileName)

		dir := filepath.Dir(destFileName)

		err = os.MkdirAll(dir, 0700) // os.ModePerm)
		if err != nil {
			return err
		}

		newFile, err := os.Create(destFileName)
		if err != nil {
			return err
		}

		defer newFile.Close()
		if err = os.Chmod(destFileName, rwxrwxrwx); err != nil {
			return err
		}

		if _, err = io.Copy(newFile, file); err != nil {
			return err
		}

	}

	return nil
}

func inputPassword(pass *string) error {
	bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err == nil {
		*pass = string(bytePassword)
		return nil
	}
	return err
}

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

	err = destFile.Chmod(rw_rw_rw_)
	if err != nil {
		return err
	}

	err = tmpl.Execute(destFile, data)
	if err != nil {
		return err
	}

	return nil
}

func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.Chmod(dest, sourceInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
