/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package istorage

import (
	"testing"
)

func TestMemTCK(t *testing.T) {
	TechnologyCompatibilityKit(t, ProvideMem())
}
