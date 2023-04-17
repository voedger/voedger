/*
 * Copyright (c) 2020-present unTill Pro, Ltd. and Contributors
 */

package edger

import (
	"time"
)

// DefaultAchieveAttemptInterval is default time interval in milliseconds between achieving attempts if first attempt has finished with errors.
var DefaultAchieveAttemptInterval time.Duration = 500 * time.Millisecond
