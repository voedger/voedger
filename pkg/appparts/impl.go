/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import "context"

type appPartitions struct {
}

func (ap appPartitions) Prepare() (err error) {
	return err
}

func (ap appPartitions) Run(ctx context.Context) {
	<-ctx.Done()
}
