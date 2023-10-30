/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

func New() (ap IAppPartitions, cleanup func(), err error) {
	return &appPartitions{}, cleanup, err
}
