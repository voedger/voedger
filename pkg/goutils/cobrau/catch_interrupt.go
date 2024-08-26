/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package cobrau

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func ExecCommandAndCatchInterrupt(cmd *cobra.Command) error {

	cmdExec := func(ctx context.Context) (err error) {
		err = cmd.ExecuteContext(ctx)
		return
	}

	err := goAndCatchInterrupt(cmdExec)
	return err
}

func goAndCatchInterrupt(f func(ctx context.Context) error) (err error) {

	var signals = make(chan os.Signal, 1)

	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(signals, os.Interrupt)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = f(ctx)
		cancel()
	}()

	select {
	case sig := <-signals:
		logger.Info("signal received:", sig)
		cancel()
	case <-ctx.Done():
	}
	logger.Verbose("waiting for function to finish...")
	wg.Wait()
	return err
}
