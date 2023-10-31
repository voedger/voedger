/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

func New() (ap IAppPartitions, cleanup func(), err error) {
	return newAppPartitions()
}

func NewAPI(ap IAppPartitions) (IAppPartitionsAPI, error) {
	return newAppPartitionsAPI(ap)
}
