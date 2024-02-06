/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

	if len(se.sshKeyPath) > 0 {
		buf := []string{}
		for _, s := range args {
			if strings.Contains(s, " ") {
				buf = append(buf, fmt.Sprintf(`"%s"`, s))
			} else {
				buf = append(buf, s)
			}
		}
		args = append([]string{fmt.Sprintf("eval $(ssh-agent -s); ssh-add %s; ./%s", se.sshKeyPath, scriptName)}, buf...)

		pExec = new(exec.PipedExec).Command("bash", "-c", strings.Join(args, " "))
	} else {
		args = append([]string{scriptName}, args...)
		pExec = new(exec.PipedExec).Command("bash", args...)
	}

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

	// nolint
	os.Chdir(scriptsTempDir)

	err := createScriptsTempDir()
	if err != nil {
		return err
	}

	// If scriptfilenames is empty, then we will copy all scripts from scriptsfs
	if len(scriptFileNames) == 0 {
		err = extractAllScripts()
		if err != nil {
			loggerError(err.Error())
			return err
		}
		return nil
	}

	for _, fileName := range scriptFileNames {

		if scriptExists(fileName) {
			continue
		}

		file, err := scriptsFS.Open("./scripts/drafts/" + fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		destFileName := filepath.Join(scriptsTempDir, fileName)

		dir := filepath.Dir(destFileName)

		// nolint
		err = os.MkdirAll(dir, rwxrwxrwx) // os.ModePerm)
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

// save all the embedded scripts into the temporary folder
func extractAllScripts() error {
	return fs.WalkDir(scriptsFS, "scripts/drafts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			content, err := fs.ReadFile(scriptsFS, path)
			if err != nil {
				return err
			}
			destPath := filepath.Join(scriptsTempDir, strings.TrimPrefix(path, "scripts/drafts"))
			err = os.MkdirAll(filepath.Dir(destPath), rwxrwxrwx)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(destPath, content, rwxrwxrwx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// nolint
func inputPassword(pass *string) error {

	bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
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

// nolint
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

// nolint
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
