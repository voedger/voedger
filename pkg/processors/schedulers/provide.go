/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package schedulers

import "github.com/voedger/voedger/pkg/appparts"

func ProvideSchedulers(cfg BasicSchedulerConfig) appparts.ISchedulerRunner {
	return newSchedulers(cfg)
}
