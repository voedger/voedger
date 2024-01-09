/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/ihttpctl"
)

type WiredServer struct {
	ihttp.IHTTPProcessor
	ihttpctl.IHTTPProcessorController
}
