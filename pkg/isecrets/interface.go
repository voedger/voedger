/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package isecrets

type ISecretReader interface {
	ReadSecret(name string) (bb []byte, err error)
}
