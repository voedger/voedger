/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/mod/modfile"
)

type goBasedDependencyManager struct {
	dependencyCachePath string
	dependencyFilePath  string
}

func (g *goBasedDependencyManager) DependencyCachePath() string {
	return g.dependencyCachePath
}

func (g *goBasedDependencyManager) GetDependencyVersion(depURL string) (string, error) {
	modFile, err := readGoModFile(g.dependencyFilePath)
	if err != nil {
		return "", err
	}

	// Find the dependency in the require statements
	for _, r := range modFile.Require {
		if r.Mod.Path == depURL {
			return r.Mod.Version, nil
		}
	}
	return "", fmt.Errorf("dependency %s not found in go.mod", depURL)
}

func (g *goBasedDependencyManager) ValidateDependencySubDir(depURL, version, subDir string) (depPath string, err error) {
	depPath = path.Join(g.dependencyCachePath, fmt.Sprintf("%s@%s", depURL, version), subDir)
	if _, err := os.Stat(depPath); os.IsNotExist(err) {
		if err := downloadDependencies(g.dependencyFilePath); err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(depPath); os.IsNotExist(err) {
		return "", err
	}
	return depPath, nil
}

func readGoModFile(dependencyFilePath string) (*modfile.File, error) {
	// TODO: checkout behaviour of modfile if we got replace in go.mod
	content, err := os.ReadFile(dependencyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s file - %v", goModFile, err)
	}

	modFile, err := modfile.Parse(dependencyFilePath, content, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s file - %v", goModFile, err)
	}
	return modFile, nil
}

func (g *goBasedDependencyManager) ParseDepQPN(depQPN string) (depURL, subDir, depVersion string, err error) {
	modFile, err := readGoModFile(g.dependencyFilePath)
	if err != nil {
		return "", "", "", err
	}

	for _, r := range modFile.Require {
		if strings.HasPrefix(depQPN, r.Mod.Path) && depQPN[len(r.Mod.Path)] == '/' {
			depURL = r.Mod.Path
			subDir = depQPN[len(r.Mod.Path)+1:]
			depVersion = r.Mod.Version
			return
		}
		if depQPN == r.Mod.Path {
			depURL = r.Mod.Path
			depVersion = r.Mod.Version
			return
		}
	}
	err = fmt.Errorf("unknown dependency %s", depQPN)
	return
}

func downloadDependencies(dependencyFilePath string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	if err := os.Chdir(path.Base(dependencyFilePath)); err != nil {
		return err
	}
	return execGoCmd("mod", "download").Err
}

func checkGoInstalled() error {
	// Check if the "go" executable is in the PATH
	if _, err := exec.LookPath("go"); err != nil {
		// Provide a more informative error message
		return fmt.Errorf("go is not installed or not in the PATH. please install Go https://golang.org/doc/install. Error - %w", err)
	}

	// Optionally, you can check the Go version to ensure it meets your application's requirements
	goVersionOutput, versionErr := execGoCmd("version").Output()
	if versionErr != nil {
		return fmt.Errorf("unable to determine Go version. Error - %v", versionErr)
	}

	// Extract the version information from the output
	versionLine := strings.Split(string(goVersionOutput), " ")[2] // Assuming the version is the third element
	goVersion := strings.TrimSpace(strings.TrimSuffix(versionLine, "\n"))
	goVersion = strings.TrimPrefix(goVersion, "go")
	if goVersion == "" {
		return fmt.Errorf("failed to extract go version from 'go version' output")
	}
	return nil
}

func locatePackageCache() (string, error) {
	goPath, ok := os.LookupEnv("GOPATH")
	if !ok {
		return "", fmt.Errorf("GOPATH env var is not defined")
	}
	return path.Join(goPath, "pkg", "mod"), nil
}

func locateDependencyFile(wd string) (string, error) {
	depFilepath := path.Join(wd, goModFile)
	if _, err := os.Stat(depFilepath); os.IsNotExist(err) {
		return "", fmt.Errorf("dependency file %s not found", goModFile)
	}
	return depFilepath, nil
}

func execGoCmd(args ...string) *exec.Cmd {
	return exec.Command("go", args...)
}
