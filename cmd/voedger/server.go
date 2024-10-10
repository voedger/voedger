/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	voedger "github.com/voedger/voedger/cmd/voedger/voedgerimpl"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/iservices"
	"github.com/voedger/voedger/pkg/iservicesctl"
)

func newServerCmd() *cobra.Command {
	var httpCLIParams ihttp.CLIParams
	var appsCLIParams voedger.CLIParams
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start server",
		RunE: func(cmd *cobra.Command, args []string) error {
			wired, cleanup, err := wireServer(httpCLIParams, appsCLIParams)
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
	serverCmd.Flags().StringVar(&appsCLIParams.Storage, "storage", "", "")
	serverCmd.Flags().StringArrayVar((*[]string)(&httpCLIParams.AcmeDomains), "acme-domain", []string{}, "")
	return serverCmd
}
