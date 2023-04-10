/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ctool

import (
	"fmt"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
)

//nolint:golint,unused
func cli(args []string, version string) int {

	// Declare and bind params

	fs := initFlags()
	bindArgsToCLIParams(fs)

	// Init and parse flags

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fs.Usage()
		return 1
	}

	cmd := fs.Arg(1)
	switch cmd {

	case "help":

		fs.Usage()
		return 0

	case "version":

		fmt.Println(version)
		return 0

	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
	}

	fs.Usage()
	return 1
}

// nolint:unused
func bindArgsToCLIParams(fs *flag.FlagSet) {

}

// nolint:unused
func initFlags() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	w := os.Stdout // may be os.Stderr - but not necessarily
	fs.SetOutput(w)
	fs.Usage = func() {
		fmt.Fprintf(w, `Usage:

	%s <command> [options]

Commands:

	help		print help
	cluster 	init
	version		print version

Options:

%s`, filepath.Base(os.Args[0]), fs.FlagUsages())
	}
	return fs
}
