/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package signupin

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConsts(t *testing.T) {
	exec.Command()
	require.True(t, validLoginRegexp.MatchString("ddd"))
}
