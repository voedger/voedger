/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package schedulers

func ProvideSchedulers(cfg BasicSchedulerConfig) ISchedulersService {
	return newSchedulers(cfg)
}
