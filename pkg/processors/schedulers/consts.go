/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package schedulers

import "time"

const (
	schedulerErrorDelay         = time.Second * 30
	defaultIntentsLimit         = 100
	borrowRetryDelay            = 50 * time.Millisecond
	initFailureErrorLogInterval = 30 * time.Second
)
