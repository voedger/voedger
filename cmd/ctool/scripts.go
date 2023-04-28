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

	"github.com/untillpro/goutils/exec"
	"golang.org/x/crypto/ssh/terminal"
)

//go:embed scripts/*
var scriptsFS embed.FS

var scriptsTempDir string

type scriptExecuterType struct {
	outputPrefix string
	sshKeyPath   string
}

func (se *scriptExecuterType) run(scriptName string, args ...string) error {

	var pExec *exec.PipedExec

	if len(se.sshKeyPath) > 0 {
		args = append([]string{fmt.Sprintf("eval $(ssh-agent -s); ssh-add %s; ./%s", se.sshKeyPath, scriptName)}, args...)
		pExec = new(exec.PipedExec).Command("bash", "-c", strings.Join(args, " "))
		pExec.GetCmd(0).Env = append(os.Environ(), "SSH_AUTH_SOCK="+getEnvValue1("SSH_AUTH_SOCK"), "SSH_AGENT_PID="+getEnvValue1("SSH_AGENT_PID"))
	} else {
		args = append([]string{scriptName}, args...)
		pExec = new(exec.PipedExec).Command("bash", args...)
	}

	os.Chdir(scriptsTempDir)

	var err error
	if len(se.outputPrefix) > 0 {
		sedArg := fmt.Sprintf("s/^/[%s]: /", se.outputPrefix)
		err = pExec.
			Command("sed", sedArg).
			Run(os.Stdout, os.Stderr)
	} else {
		err = pExec.
			Run(os.Stdout, os.Stderr)
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

		file, err := scriptsFS.Open("scripts/" + fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		destFileName := filepath.Join(scriptsTempDir, fileName)

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
