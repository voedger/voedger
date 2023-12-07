/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

import (
	"fmt"
	"go/build"
	"os"
	osexec "os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/mod/modfile"
)

type goImpl struct {
	cachePath     string
	goModFilePath string
	modFile       *modfile.File
}

func (g *goImpl) LocalPath(depURL string) (localDepPath string, err error) {
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("resolving dependency %s ...", depURL))
	}
	pkgURL, subDir, version, ok := g.parseDepURL(depURL)
	if !ok {
		return "", fmt.Errorf("cannot find module for path %s", depURL)
	}
	if version == "" {
		localDepPath = path.Join(filepath.Dir(g.goModFilePath), subDir)
		return
	}
	localDepPath = path.Join(g.cachePath, fmt.Sprintf("%s@%s", pkgURL, version), subDir)
	if _, err := os.Stat(localDepPath); os.IsNotExist(err) {
		if err := downloadDependencies(g.goModFilePath); err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(localDepPath); os.IsNotExist(err) {
		return "", err
	}
	return localDepPath, nil
}

func (g *goImpl) CachePath() string {
	return g.cachePath
}

func (g *goImpl) ModulePath() string {
	return g.modFile.Module.Mod.Path
}

// parseDepURL slices depURL into pkgURL, subDir and version.
// Empty version means depURL belongs to local project
func (g *goImpl) parseDepURL(depURL string) (pkgURL, subDir, version string, ok bool) {
	subDir, ok = matchDepPath(depURL, g.modFile.Module.Mod.Path)
	if ok {
		pkgURL = g.modFile.Module.Mod.Path
		return
	}
	for _, r := range g.modFile.Require {
		subDir, ok = matchDepPath(depURL, r.Mod.Path)
		if ok {
			pkgURL = r.Mod.Path
			version = r.Mod.Version
			return
		}
	}
	return
}

func parseGoModFile(goModPath string) (*modfile.File, error) {
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("parsing %s ...", goModPath))
	}
	// TODO: checkout behaviour of modfile if we got replace in go.mod
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %v", goModPath, err)
	}

	modFile, err := modfile.ParseLax(goModPath, content, nil)
	if err != nil {
		return nil, fmt.Errorf("errors parsing %s: %v", goModFile, err)
	}
	return modFile, nil
}

func matchDepPath(depURL, depPath string) (subDir string, ok bool) {
	ok = true
	if strings.HasPrefix(depURL, depPath) && depURL[len(depPath)] == '/' {
		subDir = depURL[len(depPath)+1:]
		return
	}
	if depURL == depPath {
		return
	}
	ok = false
	return
}

func downloadDependencies(goModFilePath string) (err error) {
	if logger.IsVerbose() {
		logger.Verbose("downloading dependencies...")
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	if err := os.Chdir(path.Dir(goModFilePath)); err != nil {
		return err
	}
	return new(exec.PipedExec).Command("go", "mod", "download").Run(nil, nil)
}

func checkGoInstalled() error {
	if logger.IsVerbose() {
		logger.Verbose("checking out go environment...")
	}
	// Check if the "go" executable is in the PATH
	if _, err := osexec.LookPath("go"); err != nil {
		return fmt.Errorf("go is required for this application but is not found. Please install Go from https://golang.org/doc/install")
	}
	return nil
}

func getCachePath() string {
	if logger.IsVerbose() {
		logger.Verbose("searching for cache of the packages")
	}
	return path.Join(build.Default.GOPATH, "pkg", "mod")
}

func getGoModFile() (*modfile.File, string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	var previousDir string
	for currentDir != previousDir {
		if logger.IsVerbose() {
			logger.Verbose(fmt.Sprintf("searching for %s in %s", goModFile, currentDir))
		}
		goModPath := filepath.Join(currentDir, goModFile)
		if _, err := os.Stat(goModPath); err == nil {
			modFile, err := parseGoModFile(goModPath)
			if err != nil {
				return nil, "", err
			}
			if logger.IsVerbose() {
				logger.Verbose(fmt.Sprintf("%s is located at %s", goModFile, currentDir))
			}
			return modFile, goModPath, nil
		}
		previousDir = currentDir
		currentDir = filepath.Dir(currentDir)
	}
	return nil, "", fmt.Errorf("%s not found", goModFile)
}
