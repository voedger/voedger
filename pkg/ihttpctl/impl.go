/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpctl

import (
	"context"
	"io/fs"
	"time"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/ihttp"
)

type httpProcessorController struct {
	api             ihttp.IHTTPProcessorAPI
	staticResources map[string]fs.FS
}

func (hc *httpProcessorController) Prepare() (err error) {
	return nil
}

func (hc *httpProcessorController) Run(ctx context.Context) {
	for path, fs := range hc.staticResources {
		for ctx.Err() == nil {
			logger.Info("deploying", path, "...")
			err := hc.api.DeployStaticContent(ctx, path, fs)
			if err == nil {
				logger.Info(path, "deployed")
				break
			}
			logger.Error("error deploying", path, ":", err)
			time.Sleep(time.Second)
		}
	}
	<-ctx.Done()
}
