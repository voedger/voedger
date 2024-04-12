/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/compile"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func newBuildCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [-C] [-o <archive-name>]",
		Short: "build",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if _, err := compile.CompileNoDummyApp(params.Dir); err != nil {
				return err
			}
			return build(params)
		},
	}
	cmd.SilenceErrors = true
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
	cmd.Flags().StringVarP(&params.Output, "output", "o", "", "output archive name")
	return cmd
}

func build(params *vpmParams) error {
	folderName := filepath.Base(params.Dir)
	wasmFilePath, err := execTinyGoBuild(params.Dir, folderName)
	if err != nil {
		return err
	}

	outputArchiveName := params.Output
	if outputArchiveName == "" {
		outputArchiveName = folderName
	}
	if !strings.HasSuffix(outputArchiveName, ".var") {
		outputArchiveName += ".var"
	}

	return coreutils.Zip(filepath.Join(params.Dir, outputArchiveName), []string{wasmFilePath})
}

// execTinyGoBuild builds the project using tinygo and returns the path to the resulting wasm file
func execTinyGoBuild(dir, folderName string) (wasmFilePath string, err error) {
	var stdout io.Writer
	if logger.IsVerbose() {
		stdout = os.Stdout
	}
	wasmFileName := folderName + ".wasm"
	if err := new(exec.PipedExec).Command("tinygo", "build", "--no-debug", "-o", wasmFileName, "-scheduler=none", "-opt=2", "-gc=leaking", "-target=wasi", ".").WorkingDir(dir).Run(stdout, os.Stderr); err != nil {
		return "", err
	}
	return filepath.Join(dir, wasmFileName), nil
}
