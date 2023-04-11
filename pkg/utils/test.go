/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package coreutils

import (
	"os"
	"strings"

	"golang.org/x/exp/slices"
)

func IsTest() bool {
	return strings.Contains(os.Args[0], ".test") || IsDebug()
}

func IsDebug() bool {
	return strings.Contains(os.Args[0], "__debug_bin")
}

func SkipSlowTests() bool {
	_, ok := os.LookupEnv("HEEUS_SKIP_SLOW_TESTS")
	return ok || slices.Contains(os.Args, "skip-slow-tests")
}
