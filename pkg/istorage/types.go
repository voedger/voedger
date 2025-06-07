/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istorage

import (
	"time"
)

type AppStorageDesc struct {
	SafeName SafeAppName
	Status   AppStorageStatus
	Error    string `json:",omitempty"`
}

type AppStorageStatus int

type SafeAppName struct {
	name string
}

// used in tests only
type IStorageDelaySetter interface {
	SetTestDelayGet(time.Duration)
	SetTestDelayPut(time.Duration)
}

