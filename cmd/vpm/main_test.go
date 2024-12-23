/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/exec"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
)

func TestCompileBasicUsage(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()

	require.NoError(err)

	testCases := []struct {
		name string
		dir  string
	}{
		{
			name: "simple schema with no imports",
			dir:  filepath.Join(wd, "testdata", "myapp", "mypkg1"),
		},
		{
			name: "schema importing a local package",
			dir:  filepath.Join(wd, "testdata", "myapp", "mypkg2"),
		},
		{
			name: "app schema importing voedger package",
			dir:  filepath.Join(wd, "testdata", "myapp", "app"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = execRootCmd([]string{"vpm", "compile", "-C", tc.dir}, "1.0.0")
			require.NoError(err)
		})
	}
}

func TestBaselineBasicUsage(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)

	tempTargetDir := t.TempDir()
	baselineDirName := "baseline_schemas"
	testCases := []struct {
		name                  string
		dir                   string
		expectedBaselineFiles []string
	}{
		{
			name: "simple schema with no imports",
			dir:  filepath.Join(wd, "testdata", "myapp", "mypkg1"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "userprofile.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg1", "schema1.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name: "schema importing a local package",
			dir:  filepath.Join(wd, "testdata", "myapp", "mypkg2"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "userprofile.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg1", "schema1.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg2", "schema2.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
		{
			name: "application schema using both local package and voedger",
			dir:  filepath.Join(wd, "testdata", "myapp", "app"),
			expectedBaselineFiles: []string{
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "sys.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "sys", "userprofile.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg1", "schema1.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "mypkg2", "schema2.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, pkgDirName, "app", "app.vsql"),
				filepath.Join(tempTargetDir, baselineDirName, baselineInfoFileName),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = os.RemoveAll(tempTargetDir)
			require.NoError(err)

			baselineDir := filepath.Join(tempTargetDir, baselineDirName)
			err = execRootCmd([]string{"vpm", "baseline", "-C", tc.dir, baselineDir}, "1.0.0")
			require.NoError(err)

			var actualFilePaths []string
			err = filepath.Walk(tempTargetDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					actualFilePaths = append(actualFilePaths, path)
				}
				return nil
			})
			require.NoError(err)

			require.Equal(len(tc.expectedBaselineFiles), len(actualFilePaths))
			for _, actualFilePath := range actualFilePaths {
				require.Contains(tc.expectedBaselineFiles, actualFilePath)
			}
		})
	}
}

func TestCompatBasicUsage(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)

	tempDir := t.TempDir()
	workDir := filepath.Join(wd, "testdata", "myapp", "app")
	baselineDir := filepath.Join(tempDir, "testdata", "baseline_myapp")
	err = execRootCmd([]string{"vpm", "baseline", baselineDir, "--change-dir", workDir}, "1.0.0")
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "compat", "-C", workDir, baselineDir}, "1.0.0")
	require.NoError(err)
}

func TestCompatErrors(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()
	require.NoError(err)

	tempDir := t.TempDir()
	workDir := filepath.Join(wd, "testdata", "myapp", "app")
	baselineDir := filepath.Join(tempDir, "testdata", "baseline_myapp")
	err = execRootCmd([]string{"vpm", "baseline", "-C", workDir, baselineDir}, "1.0.0")
	require.NoError(err)

	workDir = filepath.Join(wd, "testdata", "myapp_incompatible", "app")
	err = execRootCmd([]string{"vpm", "compat", "--ignore", filepath.Join(workDir, "ignores.yml"), "--change-dir", workDir, baselineDir}, "1.0.0")
	require.Error(err)
	errs := coreutils.SplitErrors(err)

	expectedErrs := []string{
		"OrderChanged: AppDef/Types/mypkg2.MyTable2/Fields/myfield3",
		"OrderChanged: AppDef/Types/mypkg2.MyTable2/Fields/myfield2",
	}
	require.Equal(len(expectedErrs), len(errs))

	for _, err := range errs {
		require.Contains(expectedErrs, err.Error())
	}
}

func TestCompileErrors(t *testing.T) {
	require := require.New(t)

	wd, err := os.Getwd()

	require.NoError(err)

	var tempDir string
	if logger.IsVerbose() {
		var err error
		tempDir, err = os.MkdirTemp("", "test_compile")
		require.NoError(err)
	} else {
		tempDir = t.TempDir()
	}

	err = coreutils.CopyDir(filepath.Join(wd, "testdata", "myapperr"), tempDir)
	require.NoError(err)
	// go up to the root of the project.
	localVoedgerDir := filepath.Join(wd, "..", "..")

	testCases := []struct {
		name                 string
		dir                  string
		expectedErrPositions []string
	}{
		{
			name: "schema1.vsql - syntax errors",
			dir:  "mypkg1",
			expectedErrPositions: []string{
				"schema1.vsql:6:25",
			},
		},
		{
			name: "schema1.vsql - syntax errors",
			dir:  "mypkg2",
			expectedErrPositions: []string{
				"schema1.vsql:6:25",
			},
		},
		{
			name: "schema4.vsql - package local name redeclared",
			dir:  "app",
			expectedErrPositions: []string{
				"schema4.vsql:5:1: local package name reg was redeclared as registry",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(tempDir, tc.dir)

			// replace the voedger package with the local one in the go.mod file
			// we use an absolute path so that we don't depend on where the test is running.
			err = new(exec.PipedExec).Command("go", "mod", "edit", "-replace", "github.com/voedger/voedger="+localVoedgerDir).WorkingDir(dir).Run(os.Stdout, os.Stderr)
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "compile", "-C", dir}, "1.0.0")
			require.Error(err)
			errMsg := err.Error()
			for _, expectedErrPosition := range tc.expectedErrPositions {
				require.Contains(errMsg, expectedErrPosition)
			}
			fmt.Println(err.Error())
		})
	}
}

func TestPkgRegistryCompile(t *testing.T) {
	t.Skip("This test is skipped because registry package doesn't have subdirectory 'wasm' with code inside it.")
	require := require.New(t)

	wd, err := os.Getwd()
	pkgDirLocalPath := wd[:strings.LastIndex(wd, filepath.FromSlash("/cmd/vpm"))] + filepath.FromSlash("/wasm")

	require.NoError(err)
	defer func() {
		_ = os.Chdir(wd)
	}()

	err = os.Chdir(pkgDirLocalPath)
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "compile", "-C", "registry"}, "1.0.0")
	require.NoError(err)
}

func TestOrmBasicUsage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	require := require.New(t)

	// uncomment this line to keep the result generated during test
	// the resulting dir will be printed
	// logger.SetLogLevel(logger.LogLevelVerbose)

	var err error
	var tempDir string
	if logger.IsVerbose() {
		tempDir, err = os.MkdirTemp("", "test_genorm")
		require.NoError(err)
	} else {
		tempDir = t.TempDir()
	}

	wd, err := os.Getwd()
	require.NoError(err)

	err = coreutils.CopyDir(filepath.Join(wd, "testdata", "genorm"), tempDir)
	require.NoError(err)

	tests := []struct {
		dir string
	}{
		{
			dir: "app",
		},
		{
			dir: "mypkg1",
		},
		{
			dir: "mypkg2",
		},
		{
			dir: "mypkg3",
		},
		{
			dir: "mypkg4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.dir, func(t *testing.T) {
			dir := filepath.Join(tempDir, tc.dir)
			if logger.IsVerbose() {
				logger.Verbose("------------------------------------------------------------------------")
				logger.Verbose("test dir: " + filepath.Join(dir, wasmDirName, ormDirName))
			}

			headerFile := filepath.Join(dir, "header.txt")
			err = execRootCmd([]string{"vpm", "orm", "-C", dir, "--header-file", headerFile}, "1.0.0")
			require.NoError(err)

			if logger.IsVerbose() {
				logger.Verbose("orm directory: " + filepath.Join(dir, wasmDirName, ormDirName))
				logger.Verbose("------------------------------------------------------------------------")
			}
		})
	}

}

func TestBuildExample2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	require := require.New(t)

	err := execRootCmd([]string{"vpm", "orm", "-C", "../../examples/airs-bp2/air"}, "1.0.0")
	require.NoError(err)

	wd, err := os.Getwd()
	require.NoError(err)

	airVarFile := filepath.Join(wd, "../../examples/airs-bp2/air/air.var")
	exists, err := coreutils.Exists(airVarFile)
	require.NoError(err)
	if exists {
		require.NoError(os.Remove(airVarFile))
	}

	err = execRootCmd([]string{"vpm", "build", "-C", "../../examples/airs-bp2/air"}, "1.0.0")
	require.NoError(err)
}

func TestInitBasicUsage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)

	packagePath := "github.com/account/repo"

	// test minimal required go version in normal case
	dir := t.TempDir()
	minimalRequiredGoVersionValue = "1.12"
	err := execRootCmd([]string{"vpm", "init", "-C", dir, packagePath}, "1.0.0")
	require.NoError(err)
	require.FileExists(filepath.Join(dir, goModFileName))
	require.FileExists(filepath.Join(dir, goSumFileName))
	require.FileExists(filepath.Join(dir, packagesGenFileName))
	require.DirExists(filepath.Join(dir, wasmDirName))

	// test unsupported go version
	dir = t.TempDir()
	minimalRequiredGoVersionValue = "9.99.999"
	defer func() {
		minimalRequiredGoVersionValue = minimalRequiredGoVersion
	}()
	err = execRootCmd([]string{"vpm", "init", "-C", dir, packagePath}, "1.0.0")
	require.Error(err)
	require.Contains(err.Error(), "unsupported go version")
}

func TestTidyBasicUsage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)
	logger.SetLogLevel(logger.LogLevelVerbose)
	defer logger.SetLogLevel(logger.LogLevelInfo)

	var err error
	var tempDir string
	if logger.IsVerbose() {
		tempDir, err = os.MkdirTemp("", "test_genorm")
		require.NoError(err)
	} else {
		tempDir = t.TempDir()
	}

	wd, err := os.Getwd()
	require.NoError(err)

	err = coreutils.CopyDir(filepath.Join(wd, "testdata", "build"), tempDir)
	require.NoError(err)

	dir := filepath.Join(tempDir, "appcomplex")

	err = execRootCmd([]string{"vpm", "tidy", "-C", dir}, "1.0.0")
	require.NoError(err)
}

func TestEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)

	err := execRootCmd([]string{"vpm", "tidy", "unknown"}, "1.0.0")
	require.Error(err)
	require.Equal("'vpm tidy' accepts no arguments", err.Error())

	err = execRootCmd([]string{"vpm", "tidy", "help"}, "1.0.0")
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "tidy", "help", "adads"}, "1.0.0")
	require.Error(err)
	require.Equal("'help' accepts no arguments", err.Error())

	err = execRootCmd([]string{"vpm", "init", "help"}, "1.0.0")
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "compat", "1", "2"}, "1.0.0")
	require.Error(err)
}

func TestBuildBasicUsage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	require := require.New(t)
	var tempDir string
	if logger.IsVerbose() {
		var err error
		tempDir, err = os.MkdirTemp("", "test_build")
		require.NoError(err)
	} else {
		tempDir = t.TempDir()
	}

	wd, err := os.Getwd()
	require.NoError(err)

	// go up to the root of the project.
	localVoedgerDir := filepath.Join(wd, "..", "..")

	err = coreutils.CopyDir(filepath.Join(wd, "testdata", "build"), tempDir)
	require.NoError(err)

	testCases := []struct {
		dir               string
		errMsg            string
		expectedWasmFiles []string
	}{
		{
			dir:               "noappschema",
			errMsg:            "failed to build, app schema not found",
			expectedWasmFiles: nil,
		},
		{
			dir:               "nopackagesgen",
			errMsg:            fmt.Sprintf("%s not found. Run 'vpm init'", packagesGenFileName),
			expectedWasmFiles: nil,
		},
		{
			dir:               "appsimple",
			errMsg:            "",
			expectedWasmFiles: []string{fmt.Sprintf("%s/appsimple/appsimple.wasm", buildDirName)},
		},
		{
			dir:               "appcomplex",
			errMsg:            "",
			expectedWasmFiles: []string{buildDirName + "/appcomplex/appcomplex.wasm"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.dir, func(t *testing.T) {
			dir := filepath.Join(tempDir, tc.dir)

			// replace the voedger package with the local one in the go.mod file
			// we use an absolute path so that we don't depend on where the test is running.
			err = new(exec.PipedExec).Command("go", "work", "edit", "-replace", "github.com/voedger/voedger="+localVoedgerDir).WorkingDir(tempDir).Run(os.Stdout, os.Stderr)
			require.NoError(err)

			err = execRootCmd([]string{"vpm", "build", "-C", dir, "-o", "qwerty"}, "1.0.0")
			if err != nil {
				require.Equal(tc.errMsg, err.Error())
			} else {
				require.FileExists(filepath.Join(dir, "qwerty.var"))
				err = os.Mkdir(filepath.Join(dir, "unzipped"), coreutils.FileMode_rwxrwxrwx)
				require.NoError(err)

				err = coreutils.Unzip(filepath.Join(dir, "qwerty.var"), filepath.Join(dir, "unzipped"))
				require.NoError(err)
				wasmFiles := findWasmFiles(filepath.Join(dir, "unzipped", buildDirName))
				require.Equal(len(tc.expectedWasmFiles), len(wasmFiles))
				for _, expectedWasmFile := range tc.expectedWasmFiles {
					require.Contains(wasmFiles, filepath.Join(dir, "unzipped", expectedWasmFile))
				}
			}
		})
	}
}

func TestGenOrmTestItAndBuildApp(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	require := require.New(t)
	var tempDir string
	if logger.IsVerbose() {
		var err error
		tempDir, err = os.MkdirTemp("", "test_build")
		require.NoError(err)
	} else {
		tempDir = t.TempDir()
	}

	wd, err := os.Getwd()
	require.NoError(err)

	err = coreutils.CopyDir(filepath.Join(wd, "testdata", "build"), tempDir)
	require.NoError(err)

	// test runs in the temp directory
	tempAirDir := filepath.Join(tempDir, "air")

	// go up to the root of the project.
	localVoedgerDir := filepath.Join(wd, "..", "..")

	// replace the voedger package with the local one in the go.mod file
	// we use an absolute path so that we don't depend on where the test is running.
	err = new(exec.PipedExec).Command("go", "mod", "edit", "-replace", "github.com/voedger/voedger="+localVoedgerDir).WorkingDir(tempAirDir).Run(os.Stdout, os.Stderr)
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "orm", "-C", tempAirDir}, "1.0.0")
	require.NoError(err)

	err = new(exec.PipedExec).Command("go", "test", filepath.Join(tempAirDir, "wasm")).WorkingDir(tempAirDir).Run(os.Stdout, os.Stderr)
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "build", "-C", tempAirDir}, "1.0.0")
	require.NoError(err)
}

func findWasmFiles(dir string) []string {
	var wasmFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".wasm") {
			wasmFiles = append(wasmFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return wasmFiles
}

func TestCommandMessaging(t *testing.T) {
	if testing.Short() {
		t.Skip("Manual run only because of long time execution (e.g. go get github.com/voedger/voedger run is involved)")
	}

	dir := t.TempDir()

	testCases := []testingu.CmdTestCase{
		{
			Name:                "init: wrong number of arguments",
			Args:                []string{"vpm", "init", "-C", dir, "package_path", "sfs"},
			ExpectedErrPatterns: []string{"1 arg(s)"},
		},
		{
			Name:                "init: unknown flag",
			Args:                []string{"vpm", "init", "-C", dir, "--unknown_flag", "package_path"},
			ExpectedErrPatterns: []string{"unknown flag"},
		},
		{
			Name:        "tidy: before init",
			Args:        []string{"vpm", "tidy", "-C", dir},
			ExpectedErr: errGoModFileNotFound,
		},
		{
			Name:                   "init: new package",
			Args:                   []string{"vpm", "init", "-C", dir, "package_path"},
			ExpectedStderrPatterns: []string{"go: added github.com/voedger/voedger"},
		},
		{
			Name:                   "tidy: after init",
			Args:                   []string{"vpm", "tidy", "-C", dir},
			ExpectedStdoutPatterns: []string{"failed to compile, will try to exec 'go mod tidy"},
		},
		{
			Name:                   "help",
			Args:                   []string{"vpm", "help"},
			ExpectedStdoutPatterns: []string{"vpm [command]"},
		},
		{
			Name:                   "unknown command",
			Args:                   []string{"vpm", "unknown_command"},
			ExpectedStdoutPatterns: []string{"vpm [command]"},
		},
	}

	testingu.RunCmdTestCases(t, execRootCmd, testCases, version)
}
