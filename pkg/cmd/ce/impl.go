/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ce

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/voedger/pkg/ibus"
	"github.com/untillpro/voedger/pkg/ihttp"
	"github.com/untillpro/voedger/pkg/iservices"
	"github.com/untillpro/voedger/pkg/iservicesctl"
)

func initFlags() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	w := os.Stdout // may be os.Stderr - but not necessarily
	fs.SetOutput(w)
	fs.Usage = func() {
		fmt.Fprintf(w, `Usage:

	%s <command> [options]

Commands:

	help		print help
	server		start server
	version		print version

Options:

%s`, filepath.Base(os.Args[0]), fs.FlagUsages())
	}
	return fs
}

func cli(args []string, version string) int {

	// Declare and bind params

	var busCLIParams ibus.CLIParams
	var httpCLIParams ihttp.CLIParams
	fs := initFlags()
	bindArgsToCLIParams(fs, &busCLIParams, &httpCLIParams)

	// Init and parse flags

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fs.Usage()
		return 1
	}

	cmd := fs.Arg(1)
	switch cmd {
	case "version":

		fmt.Println(version)
		return 0

	case "server":

		wired, cleanup, err := wireServer(busCLIParams, httpCLIParams)
		if err != nil {
			fmt.Fprintln(os.Stderr, "services not wired:", err)
			return 1
		}
		services := iservices.WiredStructPtrToMap(&wired)
		defer cleanup()
		runServices(services)
		return 0

	case "help":
		fs.Usage()
		return 0
	default:

		fmt.Fprintln(os.Stderr, "unknown command:", cmd)

	}

	fs.Usage()
	return 1
}

func bindArgsToCLIParams(fs *flag.FlagSet, busCLIParams *ibus.CLIParams, httpCLIParams *ihttp.CLIParams) {

	//ibus.CLIParams
	{
		fs.IntVar(&httpCLIParams.Port, "ihttp.Port", Default_ihttp_Port, "")
	}

	// ihttp.CLIParams
	{
		fs.IntVar(&busCLIParams.MaxNumOfConcurrentRequests, "ibus.MaxNumOfConcurrentRequests", Default_ibus_MaxNumOfConcurrentRequests, "")
		busCLIParams.ReadWriteTimeout = time.Nanosecond * Default_ibus_ReadWriteTimeoutNS
		if logger.IsVerbose() {
			busCLIParams.ReadWriteTimeout = time.Hour
		}
	}
}

var signals = make(chan os.Signal, 1)

func runServices(services map[string]iservices.IService) {

	signal.Notify(signals, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	ctl := iservicesctl.New()
	join, err := ctl.PrepareAndRun(ctx, services)
	if err != nil {
		cancel()
		fmt.Println("services preparation error:", err)
		return
	}
	defer join(ctx)

	sig := <-signals
	logger.Info("signal received:", sig)
	cancel()
}
