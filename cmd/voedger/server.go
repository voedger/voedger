/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/ibus"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/iservices"
	"github.com/voedger/voedger/pkg/iservicesctl"
)

func newServerCmd() *cobra.Command {
	var httpCLIParams ihttp.CLIParams
	var busCLIParams ibus.CLIParams
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start server",
		RunE: func(cmd *cobra.Command, args []string) error {
			busCLIParams.ReadWriteTimeout = time.Nanosecond * Default_ibus_ReadWriteTimeoutNS
			if logger.IsVerbose() {
				busCLIParams.ReadWriteTimeout = time.Hour //FIXME: remove this
			}
			wired, cleanup, err := wireServer(busCLIParams, httpCLIParams)
			if err != nil {
				return fmt.Errorf("services not wired: %w", err)
			}
			defer cleanup()
			services := iservices.WiredStructPtrToMap(&wired)

			ctl := iservicesctl.New()
			join, err := ctl.PrepareAndRun(cmd.Context(), services)
			if err != nil {
				return fmt.Errorf("services preparation error: %w", err)
			}
			defer join(cmd.Context())
			return nil
		},
	}
	serverCmd.PersistentFlags().IntVar(&httpCLIParams.Port, "ihttp.Port", Default_ihttp_Port, "")
	serverCmd.PersistentFlags().IntVar(&busCLIParams.MaxNumOfConcurrentRequests, "ibus.MaxNumOfConcurrentRequests", Default_ibus_MaxNumOfConcurrentRequests, "")
	return serverCmd
}
