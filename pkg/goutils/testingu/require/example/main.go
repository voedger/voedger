/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package main

import (
	"errors"
	"fmt"
)

var CrazyError = fmt.Errorf("🤪 error: %w", errors.ErrUnsupported)

func GoCrazy() { panic(CrazyError) }

func main() {
	GoCrazy()
}
