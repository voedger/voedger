/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package builtin

import iextsse "github.com/voedger/voedger/pkg/iextsse"

type LRUCache struct {
	// MaxSize in bytes
	MaxSize int
}

type Param struct {
	LRUCaches map[string]LRUCache
}

func New() iextsse.ISSEVvmFactory {
	return nil
}
