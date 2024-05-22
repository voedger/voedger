package main

import (
	"errors"
	"fmt"
)

var CrazyError = fmt.Errorf("🤪 error: %w", errors.ErrUnsupported)

func GoCrazy() { panic(CrazyError) }
