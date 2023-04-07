/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ctool

import "os"

func CLI(version string) int {
	return cli(os.Args, version)
}
