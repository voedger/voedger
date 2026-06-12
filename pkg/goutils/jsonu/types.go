/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package jsonu

import "fmt"

type jsonString string

type jsonStringer struct {
	fmt.Stringer
}

type jsonError struct {
	error
}
