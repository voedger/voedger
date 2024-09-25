/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package builtin

import iextsts "github.com/voedger/voedger/pkg/iextsts"

type LRUCache struct {
	// MaxSize in bytes
	MaxSize int
}

type Param struct {
	LRUCaches map[string]LRUCache
}

func New() iextsts.ISTSEngine {
	return nil
}
