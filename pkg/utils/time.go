/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package coreutils

import "time"

// https://github.com/golang/go/issues/27169
func ResetTimer(t *time.Timer, timeout time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(timeout)
}

