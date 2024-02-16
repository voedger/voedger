/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package smtp

type Cfg struct {
	Host      string
	Port      int32
	Username  string
	PwdSecret string
	From      string
}
